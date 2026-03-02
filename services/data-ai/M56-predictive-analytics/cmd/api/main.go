package main

import (
	"log"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/app/bootstrap"
)

func main() {
	if err := bootstrap.Build(); err != nil {
		log.Fatalf("M56-Predictive-Analytics API failed: %v", err)
	}
}
