package storage

import (
	"Hippocampus/src/client"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	efsPath      string
	s3Bucket     string
	region       string
	clients      map[string]*client.Client
	clientsMutex sync.RWMutex
	s3Sync       *S3Sync
}

func NewManager(efsPath, s3Bucket, region string) (*Manager, error) {
	if err := os.MkdirAll(efsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create EFS directory: %w", err)
	}

	s3Sync, err := NewS3Sync(s3Bucket, region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 sync: %w", err)
	}

	return &Manager{
		efsPath:  efsPath,
		s3Bucket: s3Bucket,
		region:   region,
		clients:  make(map[string]*client.Client),
		s3Sync:   s3Sync,
	}, nil
}

func (m *Manager) getClient(agentID string) (*client.Client, error) {
	m.clientsMutex.RLock()
	if c, ok := m.clients[agentID]; ok {
		m.clientsMutex.RUnlock()
		return c, nil
	}
	m.clientsMutex.RUnlock()

	m.clientsMutex.Lock()
	defer m.clientsMutex.Unlock()

	if c, ok := m.clients[agentID]; ok {
		return c, nil
	}

	filePath := filepath.Join(m.efsPath, fmt.Sprintf("%s.bin", agentID))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := m.s3Sync.DownloadIfExists(agentID, filePath); err != nil {
			return nil, fmt.Errorf("failed to download from S3: %w", err)
		}
	}

	c, err := client.New(filePath, m.region)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	m.clients[agentID] = c
	return c, nil
}

func (m *Manager) Insert(agentID, key, text string) error {
	c, err := m.getClient(agentID)
	if err != nil {
		return err
	}

	if err := c.Insert(key, text); err != nil {
		return err
	}

	filePath := filepath.Join(m.efsPath, fmt.Sprintf("%s.bin", agentID))
	go m.s3Sync.Upload(agentID, filePath)

	return nil
}

func (m *Manager) Search(agentID, text string, epsilon float32, threshold float32, topK int) (interface{}, error) {
	c, err := m.getClient(agentID)
	if err != nil {
		return nil, err
	}
	return c.Search(text, epsilon, threshold, topK)
}

func (m *Manager) InsertCSV(agentID, csvFile string) error {
	c, err := m.getClient(agentID)
	if err != nil {
		return err
	}

	if err := c.InsertCSV(csvFile); err != nil {
		return err
	}

	filePath := filepath.Join(m.efsPath, fmt.Sprintf("%s.bin", agentID))
	go m.s3Sync.Upload(agentID, filePath)

	return nil
}
