package memory

import (
	"encoding/json"
	"os"
)

type Memory struct {
	ID  string `json: "id"`
	Text string `json: "text"`
	Embedding []float32 `json: "embedding"`
}

const MemoryFile = "memories.json"

func SaveMemory(m Memory) error{
	memories, _ := LoadAllMemories()
	memories = append(memories, m)

	data, err := json.MarshalIndent(memories, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(MemoryFile, data, 0644)
}

func LoadAllMemories() ([]Memory, error){
	if _, err := os.Stat(MemoryFile); os.IsNotExist(err){
		return []Memory{}, nil
	}

	data, err := os.ReadFile(MemoryFile)
	if err != nil{
		return nil, err
	}

	var memories []Memory
	if err := json.Unmarshal(data, &memories); err != nil{
		return nil, err
	}

	return memories, nil
}