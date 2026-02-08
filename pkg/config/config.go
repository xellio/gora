package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Settings Settings `yaml:"config"`
}

type Settings struct {
	OllamaURL                   string `yaml:"ollama_url"`
	OllamaModelEmbed            string `yaml:"ollama_model_embed"`
	OllamaModel                 string `yaml:"ollama_model"`
	RedisURL                    string `yaml:"redis_url"`
	RedisIndexName              string `yaml:"redis_index_name"`
	RedisChunkSize              int    `yaml:"redis_chunk_size"`
	RedisChunkOverlap           int    `yaml:"redis_chunk_overlap"`
	DataRootPath                string `yaml:"data_root_path"`
	Debug                       bool   `yaml:"debug"`
	MaxHistoryMessages          int    `yaml:"max_history_messages"`
	AppendEmbedModelNameToIndex bool   `yaml:"append_embed_model_name_to_index"`
}

func NewDefaultConfig() *Config {
	return &Config{
		Settings: Settings{
			OllamaURL:                   "http://127.0.0.1:11434",
			OllamaModel:                 "gpt-oss:20b",
			OllamaModelEmbed:            "nomic-embed-text",
			RedisURL:                    "redis://localhost:6379",
			RedisIndexName:              "gora-doc",
			RedisChunkSize:              500,
			RedisChunkOverlap:           50,
			DataRootPath:                "data",
			Debug:                       false,
			MaxHistoryMessages:          10,
			AppendEmbedModelNameToIndex: true,
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	config := NewDefaultConfig()
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()
	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
