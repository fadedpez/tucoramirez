package game

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// ElasticsearchConfig holds configuration options for the Elasticsearch repository
type ElasticsearchConfig struct {
	URL             string
	Username        string
	Password        string
	IndexPrefix     string
	ArchivePath     string        // Path where archived game results will be stored
	RetentionPeriod time.Duration // How long to keep game results in Elasticsearch
	RotationPeriod  time.Duration // How often to rotate indices (e.g., monthly, quarterly)
	BatchSize       int           // Batch size for bulk operations
	WarmPhase       time.Duration // When to move index to warm storage (less frequently accessed)
	ColdPhase       time.Duration // When to move index to cold storage (rarely accessed)
	BackupEnabled   bool          // Whether to enable automatic backups
	BackupSchedule  string        // Cron-like schedule for backups (e.g., "0 0 * * *" for daily at midnight)
	BackupPath      string        // Path where backups will be stored
}

// DefaultElasticsearchConfig returns a default configuration for Elasticsearch
func DefaultElasticsearchConfig() *ElasticsearchConfig {
	return &ElasticsearchConfig{
		URL:             "http://localhost:9200",
		IndexPrefix:     "tucoramirez",
		ArchivePath:     "./archives",
		RetentionPeriod: 90 * 24 * time.Hour, // 90 days
		RotationPeriod:  30 * 24 * time.Hour, // 30 days (monthly)
		BatchSize:       100,
		WarmPhase:       30 * 24 * time.Hour, // 30 days
		ColdPhase:       60 * 24 * time.Hour, // 60 days
		BackupEnabled:   false,
		BackupSchedule:  "0 0 * * *", // Daily at midnight
		BackupPath:      "./backups",
	}
}

// ElasticsearchRepository implements the Repository interface using Elasticsearch
type ElasticsearchRepository struct {
	baseRepo         Repository
	client           *elasticsearch.Client
	config           *ElasticsearchConfig
	indexPrefix      string
	currentGameIndex string
	lastRotation     time.Time
}

// NewElasticsearchRepository creates a new Elasticsearch repository
func NewElasticsearchRepository(baseRepo Repository, config *ElasticsearchConfig) (*ElasticsearchRepository, error) {
	// Configure the Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{config.URL},
	}

	// Add authentication if provided
	if config.Username != "" && config.Password != "" {
		cfg.Username = config.Username
		cfg.Password = config.Password
	}

	// Create the client
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating Elasticsearch client: %w", err)
	}

	// Set default index prefix if not provided
	if config.IndexPrefix == "" {
		config.IndexPrefix = "tucoramirez"
	}

	// Set default values if not provided
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 90 * 24 * time.Hour // 90 days default
	}

	if config.RotationPeriod == 0 {
		config.RotationPeriod = 30 * 24 * time.Hour // 30 days default
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100 // Default batch size
	}

	if config.WarmPhase == 0 {
		config.WarmPhase = 30 * 24 * time.Hour // 30 days default
	}

	if config.ColdPhase == 0 {
		config.ColdPhase = 60 * 24 * time.Hour // 60 days default
	}

	// Create repository
	repo := &ElasticsearchRepository{
		baseRepo:     baseRepo,
		client:       client,
		config:       config,
		indexPrefix:  config.IndexPrefix,
		lastRotation: time.Now(),
	}

	// Initialize indices
	ctx := context.Background()
	if err := repo.initIndices(ctx); err != nil {
		return nil, fmt.Errorf("error initializing indices: %w", err)
	}

	return repo, nil
}

// initIndices creates the necessary indices if they don't exist
func (r *ElasticsearchRepository) initIndices(ctx context.Context) error {
	// Check if game index exists
	res, err := r.client.Indices.Exists([]string{r.indexPrefix + "_games"})
	if err != nil {
		return fmt.Errorf("error checking if game index exists: %w", err)
	}

	// Create game index if it doesn't exist
	if res.StatusCode == 404 {
		// Define game index mapping
		gameMapping := `{
			"mappings": {
				"properties": {
					"game_id": { "type": "keyword" },
					"game_type": { "type": "keyword" },
					"channel_id": { "type": "keyword" },
					"completed_at": { "type": "date" },
					"dealer_cards": { "type": "keyword" },
					"dealer_score": { "type": "integer" },
					"players": {
						"type": "nested",
						"properties": {
							"player_id": { "type": "keyword" },
							"hand_id": { "type": "keyword" },
							"parent_hand_id": { "type": "keyword" },
							"bet": { "type": "long" },
							"winnings": { "type": "long" },
							"result": { "type": "keyword" },
							"score": { "type": "integer" },
							"cards": { "type": "keyword" },
							"blackjack": { "type": "boolean" },
							"busted": { "type": "boolean" },
							"has_split": { "type": "boolean" },
							"is_doubled_down": { "type": "boolean" },
							"double_down_bet": { "type": "long" },
							"has_insurance": { "type": "boolean" },
							"insurance_bet": { "type": "long" },
							"insurance_payout": { "type": "long" },
							"actions": { "type": "keyword" }
						}
					}
				}
			}
		}`

		// Create game index
		req := esapi.IndicesCreateRequest{
			Index: r.indexPrefix + "_games",
			Body:  bytes.NewReader([]byte(gameMapping)),
		}

		res, err := req.Do(ctx, r.client)
		if err != nil {
			return fmt.Errorf("error creating game index: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error creating game index: %s", res.String())
		}
	}

	// Check if player index exists
	res, err = r.client.Indices.Exists([]string{r.indexPrefix + "_players"})
	if err != nil {
		return fmt.Errorf("error checking if player index exists: %w", err)
	}

	// Create player index if it doesn't exist
	if res.StatusCode == 404 {
		// Define player index mapping
		playerMapping := `{
			"mappings": {
				"properties": {
					"player_id": { "type": "keyword" },
					"game_type": { "type": "keyword" },
					"games_played": { "type": "integer" },
					"wins": { "type": "integer" },
					"losses": { "type": "integer" },
					"pushes": { "type": "integer" },
					"blackjacks": { "type": "integer" },
					"busts": { "type": "integer" },
					"splits": { "type": "integer" },
					"double_downs": { "type": "integer" },
					"insurances": { "type": "integer" },
					"total_bet": { "type": "long" },
					"total_winnings": { "type": "long" },
					"last_updated": { "type": "date" }
				}
			}
		}`

		// Create player index
		req := esapi.IndicesCreateRequest{
			Index: r.indexPrefix + "_players",
			Body:  bytes.NewReader([]byte(playerMapping)),
		}

		res, err := req.Do(ctx, r.client)
		if err != nil {
			return fmt.Errorf("error creating player index: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error creating player index: %s", res.String())
		}
	}

	return nil
}

// rotateIndices checks if it's time to rotate the indices and creates a new time-based index if needed
func (r *ElasticsearchRepository) rotateIndices(ctx context.Context) error {
	// Check if it's time to rotate the index
	if time.Since(r.lastRotation) < r.config.RotationPeriod {
		return nil
	}

	// Create a new time-based index name
	timeBasedIndex := r.indexPrefix + "_games_" + time.Now().Format("2006-01")
	
	// Check if the index already exists
	res, err := r.client.Indices.Exists([]string{timeBasedIndex})
	if err != nil {
		return fmt.Errorf("error checking if index exists: %w", err)
	}

	// Create the index if it doesn't exist
	if res.StatusCode == 404 {
		// Define game index mapping (same as in initIndices)
		gameMapping := `{
			"mappings": {
				"properties": {
					"game_id": { "type": "keyword" },
					"game_type": { "type": "keyword" },
					"channel_id": { "type": "keyword" },
					"completed_at": { "type": "date" },
					"dealer_cards": { "type": "keyword" },
					"dealer_score": { "type": "integer" },
					"players": {
						"type": "nested",
						"properties": {
							"player_id": { "type": "keyword" },
							"hand_id": { "type": "keyword" },
							"parent_hand_id": { "type": "keyword" },
							"bet": { "type": "long" },
							"winnings": { "type": "long" },
							"result": { "type": "keyword" },
							"score": { "type": "integer" },
							"cards": { "type": "keyword" },
							"blackjack": { "type": "boolean" },
							"busted": { "type": "boolean" },
							"has_split": { "type": "boolean" },
							"is_doubled_down": { "type": "boolean" },
							"double_down_bet": { "type": "long" },
							"has_insurance": { "type": "boolean" },
							"insurance_bet": { "type": "long" },
							"insurance_payout": { "type": "long" },
							"actions": { "type": "keyword" }
						}
					}
				}
			},
			"settings": {
				"number_of_shards": 1,
				"number_of_replicas": 1,
				"refresh_interval": "1s"
			}
		}`

		// Create game index
		req := esapi.IndicesCreateRequest{
			Index: timeBasedIndex,
			Body:  bytes.NewReader([]byte(gameMapping)),
		}

		res, err := req.Do(ctx, r.client)
		if err != nil {
			return fmt.Errorf("error creating time-based index: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error creating time-based index: %s", res.String())
		}

		log.Printf("Successfully created new time-based index: %s", timeBasedIndex)
	}

	// Update alias to point to the new index
	aliasName := r.indexPrefix + "_games"
	
	// Get all indices that match the pattern
	pattern := r.indexPrefix + "_games_*"
	indicesRes, err := r.client.Indices.Get([]string{pattern})
	if err != nil {
		return fmt.Errorf("error getting indices: %w", err)
	}
	defer indicesRes.Body.Close()

	// Create alias actions
	aliasActions := map[string]interface{}{
		"actions": []map[string]interface{}{
			{
				"add": map[string]interface{}{
					"index": timeBasedIndex,
					"alias": aliasName,
					"is_write_index": true,
				},
			},
		},
	}

	// Update alias
	aliasJSON, err := json.Marshal(aliasActions)
	if err != nil {
		return fmt.Errorf("error marshaling alias actions: %w", err)
	}

	req := esapi.IndicesUpdateAliasesRequest{
		Body: bytes.NewReader(aliasJSON),
	}

	res, err = req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("error updating alias: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error updating alias: %s", res.String())
	}

	// Update the current game index and last rotation time
	r.currentGameIndex = timeBasedIndex
	r.lastRotation = time.Now()

	// Check and manage lifecycle phases for all indices
	if err := r.manageIndexLifecycle(ctx); err != nil {
		log.Printf("Warning: Error managing index lifecycle: %v", err)
	}

	return nil
}

// manageIndexLifecycle checks and manages lifecycle phases for all indices
func (r *ElasticsearchRepository) manageIndexLifecycle(ctx context.Context) error {
	// Get all indices that match the pattern
	pattern := r.indexPrefix + "_games_*"
	indicesRes, err := r.client.Indices.Get([]string{pattern})
	if err != nil {
		return fmt.Errorf("error getting indices: %w", err)
	}
	defer indicesRes.Body.Close()

	// Parse the response
	var indicesInfo map[string]interface{}
	if err := json.NewDecoder(indicesRes.Body).Decode(&indicesInfo); err != nil {
		return fmt.Errorf("error parsing indices response: %w", err)
	}

	// Calculate the cutoff dates
	warmCutoffDate := time.Now().Add(-r.config.WarmPhase)
	coldCutoffDate := time.Now().Add(-r.config.ColdPhase)

	// Check each index to see if it's time to move to warm or cold storage
	for indexName := range indicesInfo {
		// Extract the date from the index name
		parts := strings.Split(indexName, "_")
		if len(parts) < 3 {
			continue
		}

		dateStr := parts[len(parts)-1]
		indexDate, err := time.Parse("2006-01", dateStr)
		if err != nil {
			log.Printf("Error parsing date from index name %s: %v", indexName, err)
			continue
		}

		// If the index is older than the warm phase, move it to warm storage
		if indexDate.Before(warmCutoffDate) {
			// Update the index settings to move it to warm storage
			settings := map[string]interface{}{
				"index": map[string]interface{}{
					"lifecycle": map[string]interface{}{
						"name": "warm_phase",
					},
					"refresh_interval": "30s", // Less frequent refresh for warm indices
					"number_of_replicas": 0,   // Reduce replicas to save space
				},
			}

			settingsJSON, err := json.Marshal(settings)
			if err != nil {
				log.Printf("Error marshaling settings for index %s: %v", indexName, err)
				continue
			}

			req := esapi.IndicesPutSettingsRequest{
				Index: []string{indexName},
				Body:  bytes.NewReader(settingsJSON),
			}

			res, err := req.Do(ctx, r.client)
			if err != nil {
				log.Printf("Error updating settings for index %s: %v", indexName, err)
				continue
			}
			defer res.Body.Close()

			if res.IsError() {
				log.Printf("Error updating settings for index %s: %s", indexName, res.String())
				continue
			}

			log.Printf("Successfully moved index %s to warm storage", indexName)
		}

		// If the index is older than the cold phase, move it to cold storage
		if indexDate.Before(coldCutoffDate) {
			// Update the index settings to move it to cold storage
			settings := map[string]interface{}{
				"index": map[string]interface{}{
					"lifecycle": map[string]interface{}{
						"name": "cold_phase",
					},
					"refresh_interval": "60s", // Even less frequent refresh for cold indices
					"number_of_replicas": 0,   // Minimize replicas for cold storage
					"blocks": map[string]interface{}{
						"write": true, // Make cold indices read-only
					},
				},
			}

			settingsJSON, err := json.Marshal(settings)
			if err != nil {
				log.Printf("Error marshaling settings for index %s: %v", indexName, err)
				continue
			}

			req := esapi.IndicesPutSettingsRequest{
				Index: []string{indexName},
				Body:  bytes.NewReader(settingsJSON),
			}

			res, err := req.Do(ctx, r.client)
			if err != nil {
				log.Printf("Error updating settings for index %s: %v", indexName, err)
				continue
			}
			defer res.Body.Close()

			if res.IsError() {
				log.Printf("Error updating settings for index %s: %s", indexName, res.String())
				continue
			}

			log.Printf("Successfully moved index %s to cold storage", indexName)
		}
	}

	return nil
}

// pruneOldIndices removes indices that are older than the retention period
func (r *ElasticsearchRepository) pruneOldIndices(ctx context.Context) error {
	// Get all indices that match the pattern
	pattern := r.indexPrefix + "_games_*"
	indicesRes, err := r.client.Indices.Get([]string{pattern})
	if err != nil {
		return fmt.Errorf("error getting indices: %w", err)
	}
	defer indicesRes.Body.Close()

	// Parse the response
	var indicesInfo map[string]interface{}
	if err := json.NewDecoder(indicesRes.Body).Decode(&indicesInfo); err != nil {
		return fmt.Errorf("error parsing indices response: %w", err)
	}

	// Calculate the cutoff date
	cutoffDate := time.Now().Add(-r.config.RetentionPeriod)

	// Check each index to see if it's older than the retention period
	for indexName := range indicesInfo {
		// Extract the date from the index name
		parts := strings.Split(indexName, "_")
		if len(parts) < 3 {
			continue
		}

		dateStr := parts[len(parts)-1]
		indexDate, err := time.Parse("2006-01", dateStr)
		if err != nil {
			log.Printf("Error parsing date from index name %s: %v", indexName, err)
			continue
		}

		// If the index is older than the retention period, archive and delete it
		if indexDate.Before(cutoffDate) {
			log.Printf("Pruning index %s (older than retention period of %v)", indexName, r.config.RetentionPeriod)
			
			// Archive the index data first
			if err := r.archiveIndex(ctx, indexName); err != nil {
				log.Printf("Error archiving index %s: %v", indexName, err)
				// Continue with deletion even if archiving fails, but only if backups are enabled
				if !r.config.BackupEnabled {
					log.Printf("Skipping deletion of index %s due to archiving failure and backups not enabled", indexName)
					continue
				}
			}

			// Create a snapshot backup if enabled
			if r.config.BackupEnabled {
				if err := r.createIndexBackup(ctx, indexName); err != nil {
					log.Printf("Error creating backup for index %s: %v", indexName, err)
					// Continue with deletion even if backup fails
				}
			}

			// Delete the index
			req := esapi.IndicesDeleteRequest{
				Index: []string{indexName},
			}

			res, err := req.Do(ctx, r.client)
			if err != nil {
				log.Printf("Error deleting index %s: %v", indexName, err)
				continue
			}
			defer res.Body.Close()

			if res.IsError() {
				log.Printf("Error deleting index %s: %s", indexName, res.String())
				continue
			}

			log.Printf("Successfully deleted index %s (older than retention period)", indexName)
		}
	}

	return nil
}

// archiveIndex archives all documents from an index to JSON files
func (r *ElasticsearchRepository) archiveIndex(ctx context.Context, indexName string) error {
	// Create the archive directory if it doesn't exist
	archiveDir := r.config.ArchivePath
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("error creating archive directory: %w", err)
	}

	// Create a query to get all documents
	queryJSON := []byte(`{
		"query": {
			"match_all": {}
		}
	}`)

	// Use the scroll API to get all documents
	// The Elasticsearch API expects a time.Duration value for the Scroll parameter
	scrollDuration := 1 * time.Minute // 1 minute scroll duration
	req := esapi.SearchRequest{
		Index:  []string{indexName},
		Body:   bytes.NewReader(queryJSON),
		Scroll: scrollDuration,
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("error searching for documents: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error searching for documents: %s", res.String())
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing search response: %w", err)
	}

	// Get the scroll ID
	scrollID, ok := result["_scroll_id"].(string)
	if !ok {
		return fmt.Errorf("scroll ID not found in search response")
	}

	// Create a deferred request to clear the scroll
	defer func() {
		clearScrollReq := esapi.ClearScrollRequest{
			ScrollID: []string{scrollID},
		}
		clearScrollRes, err := clearScrollReq.Do(ctx, r.client)
		if err != nil {
			log.Printf("Error clearing scroll: %v", err)
		}
		if clearScrollRes != nil {
			clearScrollRes.Body.Close()
		}
	}()

	// Process the hits
	hits, ok := result["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		return fmt.Errorf("hits not found in search response")
	}

	// Archive each hit
	for _, hit := range hits {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		// Get the document ID
		docID, ok := hitMap["_id"].(string)
		if !ok {
			continue
		}

		// Get the document source
		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		// Convert the source to a GameResult
		gameResult, err := r.convertToGameResult(source)
		if err != nil {
			log.Printf("Error converting document %s to GameResult: %v", docID, err)
			continue
		}

		// Create a unique identifier for the game result
		gameID := fmt.Sprintf("%s_%d", gameResult.ChannelID, gameResult.CompletedAt.UnixNano())

		// Create a file name based on the game ID
		fileName := filepath.Join(archiveDir, fmt.Sprintf("%s.json.gz", gameID))

		// Open the file for writing
		file, err := os.Create(fileName)
		if err != nil {
			log.Printf("Error creating archive file for document %s: %v", docID, err)
			continue
		}

		// Create a gzip writer
		gzipWriter := gzip.NewWriter(file)

		// Marshal the game result to JSON
		jsonData, err := json.Marshal(gameResult)
		if err != nil {
			log.Printf("Error marshaling game result for document %s: %v", docID, err)
			gzipWriter.Close()
			file.Close()
			continue
		}

		// Write the JSON data to the gzip writer
		if _, err := gzipWriter.Write(jsonData); err != nil {
			log.Printf("Error writing to gzip writer for document %s: %v", docID, err)
			gzipWriter.Close()
			file.Close()
			continue
		}

		// Close the writers
		gzipWriter.Close()
		file.Close()
	}

	// Continue scrolling until there are no more hits
	for {
		// Create a scroll request
		scrollReq := esapi.ScrollRequest{
			ScrollID: scrollID,
			Scroll:   scrollDuration,
		}

		scrollRes, err := scrollReq.Do(ctx, r.client)
		if err != nil {
			return fmt.Errorf("error scrolling: %w", err)
		}
		defer scrollRes.Body.Close()

		if scrollRes.IsError() {
			return fmt.Errorf("error scrolling: %s", scrollRes.String())
		}

		// Parse the response
		var scrollResult map[string]interface{}
		if err := json.NewDecoder(scrollRes.Body).Decode(&scrollResult); err != nil {
			return fmt.Errorf("error parsing scroll response: %w", err)
		}

		// Get the scroll ID
		scrollID, ok = scrollResult["_scroll_id"].(string)
		if !ok {
			return fmt.Errorf("scroll ID not found in scroll response")
		}

		// Process the hits
		hits, ok = scrollResult["hits"].(map[string]interface{})["hits"].([]interface{})
		if !ok {
			return fmt.Errorf("hits not found in scroll response")
		}

		// If there are no more hits, break
		if len(hits) == 0 {
			break
		}

		// Archive each hit
		for _, hit := range hits {
			hitMap, ok := hit.(map[string]interface{})
			if !ok {
				continue
			}

			// Get the document ID
			docID, ok := hitMap["_id"].(string)
			if !ok {
				continue
			}

			// Get the document source
			source, ok := hitMap["_source"].(map[string]interface{})
			if !ok {
				continue
			}

			// Convert the source to a GameResult
			gameResult, err := r.convertToGameResult(source)
			if err != nil {
				log.Printf("Error converting document %s to GameResult: %v", docID, err)
				continue
			}

			// Create a unique identifier for the game result
			gameID := fmt.Sprintf("%s_%d", gameResult.ChannelID, gameResult.CompletedAt.UnixNano())

			// Create a file name based on the game ID
			fileName := filepath.Join(archiveDir, fmt.Sprintf("%s.json.gz", gameID))

			// Open the file for writing
			file, err := os.Create(fileName)
			if err != nil {
				log.Printf("Error creating archive file for document %s: %v", docID, err)
				continue
			}

			// Create a gzip writer
			gzipWriter := gzip.NewWriter(file)

			// Marshal the game result to JSON
			jsonData, err := json.Marshal(gameResult)
			if err != nil {
				log.Printf("Error marshaling game result for document %s: %v", docID, err)
				gzipWriter.Close()
				file.Close()
				continue
			}

			// Write the JSON data to the gzip writer
			if _, err := gzipWriter.Write(jsonData); err != nil {
				log.Printf("Error writing to gzip writer for document %s: %v", docID, err)
				gzipWriter.Close()
				file.Close()
				continue
			}

			// Close the writers
			gzipWriter.Close()
			file.Close()
		}
	}

	return nil
}

// createIndexBackup creates a snapshot backup of an index
func (r *ElasticsearchRepository) createIndexBackup(ctx context.Context, indexName string) error {
	// Create backup directory if it doesn't exist
	backupDir := r.config.BackupPath
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("error creating backup directory: %w", err)
	}

	// Create a unique backup name based on the index name and timestamp
	backupName := fmt.Sprintf("%s_%s", indexName, time.Now().Format("20060102150405"))

	// Create a repository for the snapshot if it doesn't exist
	repoName := r.indexPrefix + "_backup_repo"
	repoExists, err := r.checkSnapshotRepositoryExists(ctx, repoName)
	if err != nil {
		return fmt.Errorf("error checking if snapshot repository exists: %w", err)
	}

	if !repoExists {
		// Create the repository
		repoConfig := map[string]interface{}{
			"type": "fs",
			"settings": map[string]interface{}{
				"location": backupDir,
				"compress": true,
			},
		}

		repoJSON, err := json.Marshal(repoConfig)
		if err != nil {
			return fmt.Errorf("error marshaling repository config: %w", err)
		}

		req := esapi.SnapshotCreateRepositoryRequest{
			Repository: repoName,
			Body:       bytes.NewReader(repoJSON),
		}

		res, err := req.Do(ctx, r.client)
		if err != nil {
			return fmt.Errorf("error creating snapshot repository: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error creating snapshot repository: %s", res.String())
		}

		log.Printf("Successfully created snapshot repository: %s", repoName)
	}

	// Create the snapshot
	snapshotConfig := map[string]interface{}{
		"indices": indexName,
		"ignore_unavailable": true,
		"include_global_state": false,
	}

	snapshotJSON, err := json.Marshal(snapshotConfig)
	if err != nil {
		return fmt.Errorf("error marshaling snapshot config: %w", err)
	}

	req := esapi.SnapshotCreateRequest{
		Repository: repoName,
		Snapshot:   backupName,
		Body:       bytes.NewReader(snapshotJSON),
		WaitForCompletion: func() *bool { b := true; return &b }(),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("error creating snapshot: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error creating snapshot: %s", res.String())
	}

	log.Printf("Successfully created backup snapshot %s for index %s", backupName, indexName)
	return nil
}

// checkSnapshotRepositoryExists checks if a snapshot repository exists
func (r *ElasticsearchRepository) checkSnapshotRepositoryExists(ctx context.Context, repoName string) (bool, error) {
	req := esapi.SnapshotGetRepositoryRequest{
		Repository: []string{repoName},
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return false, fmt.Errorf("error checking if snapshot repository exists: %w", err)
	}
	defer res.Body.Close()

	// If the repository doesn't exist, we'll get a 404
	if res.StatusCode == 404 {
		return false, nil
	}

	// If we got an error other than 404, return it
	if res.IsError() {
		return false, fmt.Errorf("error checking if snapshot repository exists: %s", res.String())
	}

	return true, nil
}

// convertToGameResult converts an Elasticsearch document to a GameResult
func (r *ElasticsearchRepository) convertToGameResult(source map[string]interface{}) (*entities.GameResult, error) {
	// Convert the source to JSON
	sourceJSON, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("error marshaling source: %w", err)
	}

	// Unmarshal the JSON to a GameResult
	var gameResult entities.GameResult
	if err := json.Unmarshal(sourceJSON, &gameResult); err != nil {
		return nil, fmt.Errorf("error unmarshaling source: %w", err)
	}

	return &gameResult, nil
}

// ArchiveGameResults archives game results to JSON files
func (r *ElasticsearchRepository) ArchiveGameResults(ctx context.Context, gameResults []*entities.GameResult) error {
	// Create the archive directory if it doesn't exist
	archiveDir := r.config.ArchivePath
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("error creating archive directory: %w", err)
	}

	// Create a file for each game result
	for _, result := range gameResults {
		// Create a unique identifier for the game result
		// Since GameResult doesn't have a dedicated ID field, we'll create one from channel ID and timestamp
		gameID := fmt.Sprintf("%s_%d", result.ChannelID, result.CompletedAt.UnixNano())

		// Create a file name based on the game ID
		fileName := filepath.Join(archiveDir, fmt.Sprintf("%s.json.gz", gameID))

		// Open the file for writing
		file, err := os.Create(fileName)
		if err != nil {
			return fmt.Errorf("error creating archive file: %w", err)
		}
		defer file.Close()

		// Create a gzip writer
		gzipWriter := gzip.NewWriter(file)

		// Marshal the game result to JSON
		jsonData, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("error marshaling game result: %w", err)
		}

		// Write the JSON data to the gzip writer
		if _, err := gzipWriter.Write(jsonData); err != nil {
			return fmt.Errorf("error writing to gzip writer: %w", err)
		}

		// Close the writers
		gzipWriter.Close()
	}

	return nil
}

// UpdatePlayerStatistics updates player statistics in the base repository
// For Elasticsearch, we don't need to explicitly update statistics as they're calculated on-the-fly
func (r *ElasticsearchRepository) UpdatePlayerStatistics(ctx context.Context, gameResult *entities.GameResult) error {
	// First update in the base repository
	if err := r.baseRepo.UpdatePlayerStatistics(ctx, gameResult); err != nil {
		return err
	}
	
	// For Elasticsearch, we don't need to do anything special here
	// as statistics are calculated on-the-fly when queried
	return nil
}

// Close closes the Elasticsearch client and the base repository
func (r *ElasticsearchRepository) Close() error {
	// Close the base repository
	return r.baseRepo.Close()
}

// GetPlayerStatistics retrieves player statistics from Elasticsearch
func (r *ElasticsearchRepository) GetPlayerStatistics(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	// Define the query to find the player statistics
	query := fmt.Sprintf(`{
		"query": {
			"bool": {
				"must": [
					{ "term": { "player_id": "%s" } },
					{ "term": { "game_type": "%s" } }
				]
			}
		}
	}`, playerID, gameType)

	// Search for the player statistics
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexPrefix+"_players"),
		r.client.Search.WithBody(bytes.NewReader([]byte(query))),
		r.client.Search.WithSize(1),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for player statistics: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching for player statistics: %s", res.String())
	}

	// Parse the response
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source entities.PlayerStatistics `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing player statistics: %w", err)
	}

	// If no statistics found, return empty statistics
	if result.Hits.Total.Value == 0 {
		return &entities.PlayerStatistics{
			PlayerID:    playerID,
			GameType:    gameType,
			LastUpdated: time.Now(),
		}, nil
	}

	return &result.Hits.Hits[0].Source, nil
}

// GetAllPlayerStatistics retrieves all player statistics from Elasticsearch
func (r *ElasticsearchRepository) GetAllPlayerStatistics(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	// Define the query to find all player statistics for a game type
	query := fmt.Sprintf(`{
		"query": {
			"term": { "game_type": "%s" }
		},
		"sort": [
			{ "total_winnings": { "order": "desc" } }
		]
	}`, gameType)

	// Search for player statistics
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexPrefix+"_players"),
		r.client.Search.WithBody(bytes.NewReader([]byte(query))),
		r.client.Search.WithSize(100), // Limit to 100 players
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for player statistics: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching for player statistics: %s", res.String())
	}

	// Parse the response
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source entities.PlayerStatistics `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing player statistics: %w", err)
	}

	// If no statistics found, return empty slice
	if result.Hits.Total.Value == 0 {
		return []*entities.PlayerStatistics{}, nil
	}

	// Convert to slice of player statistics
	statsList := make([]*entities.PlayerStatistics, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		statsList[i] = &hit.Source
	}

	return statsList, nil
}

// GetAllPlayerStatisticsFromES retrieves all player statistics directly from Elasticsearch
// This is a specialized method that bypasses the base repository
func (r *ElasticsearchRepository) GetAllPlayerStatisticsFromES(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	// This is essentially the same as GetAllPlayerStatistics, but we're making it explicit
	// that this method is specifically for retrieving from Elasticsearch
	return r.GetAllPlayerStatistics(ctx, gameType)
}

// GetChannelResults retrieves game results for a specific channel from Elasticsearch
func (r *ElasticsearchRepository) GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error) {
	// Define the query to find game results for the channel
	query := fmt.Sprintf(`{
		"query": {
			"term": { "channel_id": "%s" }
		},
		"sort": [
			{ "completed_at": { "order": "desc" } }
		]
	}`, channelID)

	// Search for game results
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexPrefix+"_games_*"),
		r.client.Search.WithBody(bytes.NewReader([]byte(query))),
		r.client.Search.WithSize(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for channel results: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching for channel results: %s", res.String())
	}

	// Parse the response
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing channel results: %w", err)
	}

	// If no results found, return empty slice
	if result.Hits.Total.Value == 0 {
		return []*entities.GameResult{}, nil
	}

	// Convert to slice of game results
	resultsList := make([]*entities.GameResult, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		gameResult, err := r.convertToGameResult(hit.Source)
		if err != nil {
			// Log the error but continue processing other results
			fmt.Printf("Error converting game result: %v\n", err)
			continue
		}
		resultsList = append(resultsList, gameResult)
	}

	return resultsList, nil
}

// GetDeck retrieves a deck by ID from the base repository
func (r *ElasticsearchRepository) GetDeck(ctx context.Context, deckID string) ([]*entities.Card, error) {
	// Delegate to the base repository
	return r.baseRepo.GetDeck(ctx, deckID)
}

// SaveDeck saves a deck to the base repository
func (r *ElasticsearchRepository) SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error {
	// Delegate to the base repository
	return r.baseRepo.SaveDeck(ctx, channelID, deck)
}

// GetPlayerResults retrieves game results for a specific player from Elasticsearch
func (r *ElasticsearchRepository) GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error) {
	// Define the query to find game results for the player
	query := fmt.Sprintf(`{
		"query": {
			"nested": {
				"path": "players",
				"query": {
					"term": { "players.player_id": "%s" }
				}
			}
		},
		"sort": [
			{ "completed_at": { "order": "desc" } }
		]
	}`, playerID)

	// Search for game results - limit to 20 most recent games
	limit := 20
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexPrefix+"_games_*"),
		r.client.Search.WithBody(bytes.NewReader([]byte(query))),
		r.client.Search.WithSize(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for player results: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching for player results: %s", res.String())
	}

	// Parse the response
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing player results: %w", err)
	}

	// If no results found, return empty slice
	if result.Hits.Total.Value == 0 {
		return []*entities.GameResult{}, nil
	}

	// Convert to slice of game results
	resultsList := make([]*entities.GameResult, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		gameResult, err := r.convertToGameResult(hit.Source)
		if err != nil {
			// Log the error but continue processing other results
			fmt.Printf("Error converting game result: %v\n", err)
			continue
		}
		resultsList = append(resultsList, gameResult)
	}

	return resultsList, nil
}

// GetPlayerStatisticsFromES retrieves player statistics directly from Elasticsearch
// This is a specialized method that bypasses the base repository
func (r *ElasticsearchRepository) GetPlayerStatisticsFromES(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	// Define the query to find the player statistics
	query := fmt.Sprintf(`{
		"query": {
			"bool": {
				"must": [
					{ "term": { "player_id": "%s" } },
					{ "term": { "game_type": "%s" } }
				]
			}
		}
	}`, playerID, gameType)

	// Search for the player statistics
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexPrefix+"_players"),
		r.client.Search.WithBody(bytes.NewReader([]byte(query))),
		r.client.Search.WithSize(1),
	)
	if err != nil {
		return nil, fmt.Errorf("error searching for player statistics: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching for player statistics: %s", res.String())
	}

	// Parse the response
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source entities.PlayerStatistics `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing player statistics: %w", err)
	}

	// If no statistics found, return empty statistics
	if result.Hits.Total.Value == 0 {
		return &entities.PlayerStatistics{
			PlayerID:    playerID,
			GameType:    gameType,
			LastUpdated: time.Now(),
		}, nil
	}

	return &result.Hits.Hits[0].Source, nil
}

// IndexGameResult indexes a game result in Elasticsearch
func (r *ElasticsearchRepository) IndexGameResult(ctx context.Context, gameResult *entities.GameResult) error {
	// First check if we need to rotate indices
	if err := r.rotateIndices(ctx); err != nil {
		return fmt.Errorf("error rotating indices: %w", err)
	}

	// Convert the game result to JSON
	jsonData, err := json.Marshal(gameResult)
	if err != nil {
		return fmt.Errorf("error marshaling game result: %w", err)
	}

	// Index the game result
	res, err := r.client.Index(
		r.currentGameIndex,
		bytes.NewReader(jsonData),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("error indexing game result: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing game result: %s", res.String())
	}

	return nil
}

// PruneGameResultsPerPlayer prunes game results per player, keeping only the most recent ones
func (r *ElasticsearchRepository) PruneGameResultsPerPlayer(ctx context.Context, maxMatchesPerPlayer int) error {
	// Get all unique player IDs from the game results
	query := `{
		"size": 0,
		"aggs": {
			"players": {
				"nested": { "path": "players" },
				"aggs": {
					"unique_players": {
						"terms": { "field": "players.player_id", "size": 10000 }
					}
				}
			}
		}
	}`

	// Search for unique player IDs
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexPrefix+"_games_*"),
		r.client.Search.WithBody(bytes.NewReader([]byte(query))),
	)
	if err != nil {
		return fmt.Errorf("error searching for unique player IDs: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error searching for unique player IDs: %s", res.String())
	}

	// Parse the response to get unique player IDs
	var result struct {
		Aggregations struct {
			Players struct {
				UniquePlayer struct {
					Buckets []struct {
						Key string `json:"key"`
					} `json:"buckets"`
				} `json:"unique_players"`
			} `json:"players"`
		} `json:"aggregations"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("error parsing unique player IDs: %w", err)
	}

	// For each player, get all their game results and keep only the most recent ones
	for _, bucket := range result.Aggregations.Players.UniquePlayer.Buckets {
		playerID := bucket.Key

		// Get all game results for this player
		playerResults, err := r.GetPlayerResults(ctx, playerID)
		if err != nil {
			// Log the error but continue with other players
			fmt.Printf("Error getting game results for player %s: %v\n", playerID, err)
			continue
		}

		// If the player has more than the maximum allowed matches, delete the oldest ones
		if len(playerResults) > maxMatchesPerPlayer {
			// Sort the results by completion time (newest first)
			sort.Slice(playerResults, func(i, j int) bool {
				return playerResults[i].CompletedAt.After(playerResults[j].CompletedAt)
			})

			// Keep only the most recent matches
			resultsToDelete := playerResults[maxMatchesPerPlayer:]

			// Archive the results before deleting them
			if err := r.ArchiveGameResults(ctx, resultsToDelete); err != nil {
				// Log the error but continue with deletion
				fmt.Printf("Error archiving game results for player %s: %v\n", playerID, err)
			}

			// Delete the old results
			for _, result := range resultsToDelete {
				// Create a query to find this specific game result
				deleteQuery := fmt.Sprintf(`{
					"query": {
						"bool": {
							"must": [
								{ "term": { "channel_id": "%s" } },
								{ "term": { "completed_at": "%s" } }
							]
						}
					}
				}`, result.ChannelID, result.CompletedAt.Format(time.RFC3339))

				// Delete by query
				deleteRes, err := r.client.DeleteByQuery(
					[]string{r.indexPrefix + "_games_*"},
					bytes.NewReader([]byte(deleteQuery)),
					r.client.DeleteByQuery.WithContext(ctx),
					r.client.DeleteByQuery.WithRefresh(true),
				)
				if err != nil {
					// Log the error but continue with other results
					fmt.Printf("Error deleting game result for player %s: %v\n", playerID, err)
					continue
				}
				deleteRes.Body.Close()

				if deleteRes.IsError() {
					// Log the error but continue with other results
					fmt.Printf("Error deleting game result for player %s: %s\n", playerID, deleteRes.String())
				}
			}
		}
	}

	return nil
}

// SaveGameResult saves a game result to the base repository and indexes it in Elasticsearch
func (r *ElasticsearchRepository) SaveGameResult(ctx context.Context, result *entities.GameResult) error {
	// First save to the base repository
	if err := r.baseRepo.SaveGameResult(ctx, result); err != nil {
		return fmt.Errorf("error saving game result to base repository: %w", err)
	}

	// Then index in Elasticsearch
	return r.IndexGameResult(ctx, result)
}

// RotateIndices checks if it's time to rotate the indices and creates a new time-based index if needed
func (r *ElasticsearchRepository) RotateIndices(ctx context.Context) error {
	return r.rotateIndices(ctx)
}

// PruneOldIndices removes indices that are older than the retention period
func (r *ElasticsearchRepository) PruneOldIndices(ctx context.Context) error {
	return r.pruneOldIndices(ctx)
}

// CreateIndexBackup creates a snapshot backup of an index
func (r *ElasticsearchRepository) CreateIndexBackup(ctx context.Context, indexName string) error {
	return r.createIndexBackup(ctx, indexName)
}

// GetIndices returns a list of indices that match the given pattern
func (r *ElasticsearchRepository) GetIndices(ctx context.Context, pattern string) ([]string, error) {
	res, err := r.client.Indices.Get(
		[]string{pattern},
		r.client.Indices.Get.WithContext(ctx),
		r.client.Indices.Get.WithExpandWildcards("open"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get indices: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error getting indices: %s", res.String())
	}

	// Parse the response to get the index names
	var indices map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, fmt.Errorf("error parsing indices response: %w", err)
	}

	// Extract the index names from the map keys
	indexNames := make([]string, 0, len(indices))
	for name := range indices {
		indexNames = append(indexNames, name)
	}

	return indexNames, nil
}

// GetConfig returns the repository configuration
func (r *ElasticsearchRepository) GetConfig() ElasticsearchConfig {
	return *r.config
}

// GetIndexPrefix returns the index prefix used by the repository
func (r *ElasticsearchRepository) GetIndexPrefix() string {
	return r.indexPrefix
}
