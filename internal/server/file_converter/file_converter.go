package file_converter

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type Producer struct {
	filename string
}

func NewProducer(filename string) (*Producer, error) {
	return &Producer{
		filename: filename,
	}, nil
}

func (p *Producer) WriteEvent(metricsList []models.Metrics) error {
	dir := filepath.Dir(p.filename)
	pattern := filepath.Base(p.filename) + ".tmp-*"

	tempFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}

	tempFilename := tempFile.Name()
	defer os.Remove(tempFilename)

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(metricsList); err != nil {
		tempFile.Close()
		return err
	}

	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempFilename, p.filename)
}

func (p *Producer) Close() error {
	return nil
}

type Consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(filename string) (*Consumer, error) {
	// откройте файл и создайте для него json.Decoder
	// допишите код здесь
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (c *Consumer) ReadMetricsList() ([]models.Metrics, error) {
	var metricsList []models.Metrics
	if err := c.decoder.Decode(&metricsList); err != nil {
		if errors.Is(err, io.EOF) {
			return []models.Metrics{}, nil
		}
		return nil, err
	}
	return metricsList, nil
}

func (c *Consumer) Close() error {
	return c.file.Close()
}
