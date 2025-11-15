package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ankur-anand/unisondb/dbkernel"
	"github.com/ankur-anand/unisondb/internal/services/relayer"
	"github.com/ankur-anand/unisondb/pkg/kvdrivers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	clientName = "GRPC-CLIENT-2"
	clientPort = "9002"
)

var (
	upstreamAddr = flag.String("upstream", "localhost:4001", "Upstream UnisonDB gRPC address")
	namespace    = flag.String("namespace", "demo", "Namespace to replicate")
	dataDir      = flag.String("datadir", "/tmp/unisondb-client2", "Data directory for local replica")
)

// LoggingWalHandler wraps the standard WAL handler to log incoming data
type LoggingWalHandler struct {
	underlying relayer.WalIO
	clientName string
}

func NewLoggingWalHandler(underlying relayer.WalIO, clientName string) *LoggingWalHandler {
	return &LoggingWalHandler{
		underlying: underlying,
		clientName: clientName,
	}
}

func (l *LoggingWalHandler) Write(data interface{}) error {
	// Log the incoming WAL record
	log.Printf("[%s] 📥 Received WAL record from UnisonDB", l.clientName)

	// Delegate to underlying handler
	if walIO, ok := l.underlying.(relayer.WalIO); ok {
		return walIO.Write(data)
	}
	return fmt.Errorf("underlying handler does not implement WalIO")
}

func (l *LoggingWalHandler) WriteBatch(records interface{}) error {
	// Type assertion to get the actual records
	if recs, ok := records.([]*interface{}); ok {
		log.Printf("[%s] 📦 Received batch of %d WAL records from UnisonDB", l.clientName, len(recs))
	} else {
		log.Printf("[%s] 📦 Received WAL batch from UnisonDB", l.clientName)
	}

	// Delegate to underlying handler
	if walIO, ok := l.underlying.(relayer.WalIO); ok {
		return walIO.WriteBatch(records)
	}
	return fmt.Errorf("underlying handler does not implement WalIO")
}

func main() {
	flag.Parse()

	log.Printf("🚀 Starting %s", clientName)
	log.Printf("🔗 Upstream: %s", *upstreamAddr)
	log.Printf("📁 Data directory: %s", *dataDir)
	log.Printf("📚 Namespace: %s", *namespace)
	log.Printf("🌐 Client listening on port: %s", clientPort)

	// Create data directory
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database engine for local replica
	engine, err := initializeEngine(*dataDir, *namespace)
	if err != nil {
		log.Fatalf("Failed to initialize engine: %v", err)
	}
	defer engine.Close()

	log.Printf("✅ Local replica engine initialized")

	// Setup gRPC connection to upstream UnisonDB
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := grpc.Dial(
		*upstreamAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32*1024*1024)),
	)
	if err != nil {
		log.Fatalf("Failed to connect to upstream: %v", err)
	}
	defer conn.Close()

	log.Printf("✅ Connected to upstream UnisonDB")

	// Create relayer
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	segmentLagThreshold := 10
	rel := relayer.NewRelayer(
		engine,
		*namespace,
		conn,
		segmentLagThreshold,
		logger,
	)

	log.Printf("✅ Relayer configured")

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Printf("🛑 Shutting down %s...", clientName)
		cancel()
	}()

	// Start replication
	log.Printf("🔄 Starting replication stream for namespace '%s'...", *namespace)
	log.Printf("📊 Monitoring incoming data from UnisonDB...")
	log.Println("─────────────────────────────────────────────")

	// Add a ticker to show we're alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				currentOffset := engine.CurrentOffset()
				if currentOffset != nil {
					log.Printf("[%s] 📍 Current offset: segment=%d, offset=%d",
						clientName, currentOffset.SegmentID, currentOffset.Offset)
				} else {
					log.Printf("[%s] 📍 Waiting for first record...", clientName)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start relay (this blocks until context is cancelled)
	if err := rel.StartRelay(ctx); err != nil {
		if ctx.Err() != context.Canceled {
			log.Fatalf("Relay error: %v", err)
		}
	}

	log.Printf("✅ %s shut down successfully", clientName)
}

func initializeEngine(dataDir, namespace string) (*dbkernel.Engine, error) {
	// Storage configuration
	storageConfig := dbkernel.StorageConfig{
		BaseDir:      dataDir,
		Namespaces:   []string{namespace},
		BytesPerSync: 1024 * 1024,      // 1MB
		SegmentSize:  16 * 1024 * 1024, // 16MB
	}

	// Use BoltDB as the backend
	driverFactory := kvdrivers.BoltDBDriverFactory{}

	engine, err := dbkernel.New(storageConfig, driverFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	return engine, nil
}
