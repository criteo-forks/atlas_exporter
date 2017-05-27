package main

import (
	"log"

	"errors"
	"fmt"
	"time"

	"github.com/DNS-OARC/ripeatlas"
	"github.com/DNS-OARC/ripeatlas/measurement"
	"github.com/czerwonk/atlas_exporter/dns"
	"github.com/czerwonk/atlas_exporter/metric"
	"github.com/czerwonk/atlas_exporter/ntp"
	"github.com/czerwonk/atlas_exporter/ping"
	"github.com/czerwonk/atlas_exporter/probe"
	"github.com/czerwonk/atlas_exporter/traceroute"
)

func getMeasurement(id string) ([]metric.MetricExporter, error) {
	a := ripeatlas.Atlaser(ripeatlas.NewHttp())
	c, err := a.MeasurementLatest(ripeatlas.Params{"pk": id})

	if err != nil {
		return nil, err
	}

	res := make([]metric.MetricExporter, 0)
	ch := make(chan metric.MetricExporter)

	count := 0
	for r := range c {
		if r.ParseError != nil {
			return nil, err
		}

		go getMetricExporter(r, ch)
		count++
	}

	for i := 0; i < count; i++ {
		select {
		case m := <-ch:
			if m != nil && (!*filterInvalidResults || m.IsValid()) {
				res = append(res, m)
			}
		case <-time.After(60 * time.Second):
			return nil, errors.New(fmt.Sprintln("Timeout exceeded!"))
		}
	}

	return res, nil
}

func getMetricExporter(r *measurement.Result, out chan metric.MetricExporter) {
	var m metric.MetricExporter

	if r.Type() == "ping" {
		m = ping.FromResult(r)
	}

	if r.Type() == "traceroute" {
		m = traceroute.FromResult(r)
	}

	if r.Type() == "ntp" {
		m = ntp.FromResult(r)
	}

	if r.Type() == "dns" {
		m = dns.FromResult(r)
	}

	if m != nil {
		setAsnForMetricExporter(r, m)
	} else {
		log.Printf("Type %s is not yet supported\n", r.Type())
	}

	out <- m
}
func setAsnForMetricExporter(r *measurement.Result, m metric.MetricExporter) {
	p, err := probe.Get(r.PrbId())

	if err != nil {
		log.Printf("Could not get information for probe %d: %v\n", r.PrbId(), err)
		return
	}

	if r.Af() == 4 {
		m.SetAsn(p.Asn4)
	} else {
		m.SetAsn(p.Asn6)
	}
}
