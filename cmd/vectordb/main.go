package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Kasbe14/Dattaniddhi/internal/collection"
	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func main() {
	collName := flag.String("collection", "default_collection", "Name of the collection")
	dim := flag.Int("dim", 128, "Vector dimension")
	flag.Parse()

	rootDir := filepath.Join(".", "data")

	config := collection.CollectionConfig{
		Name:      *collName,
		Dimension: *dim,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Metric:    types.Cosine,
		ModelName: "test-model",
	}

	var coll *collection.Collection

	log.Printf("Attempting to create collection '%s'...", *collName)

	createdColl, err := collection.CreateCollection(config, rootDir, wal.SyncOS)

	if err == nil {
		coll = createdColl
		log.Printf("Collection '%s' created successfully", *collName)
	} else {
		if errors.Is(err, collection.ErrCollectionAlreadyExists) {
			log.Printf("Collection already exists. Opening existing collection...")

			coll, err = collection.OpenCollection(rootDir, *collName, wal.SyncOS)
			if err != nil {
				log.Fatalf("FATAL: failed to open collection: %v", err)
			}
		} else {
			log.Fatalf("FATAL: failed to create collection: %v", err)
		}
	}

	log.Println("✅ Dattaniddhi is online and ready for traffic")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Received shutdown signal...")

	if err := coll.Close(); err != nil {
		log.Fatalf("FATAL: clean shutdown failed: %v", err)
	}

	log.Println("✅ Shutdown complete")
}
