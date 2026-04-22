package file_converter

import (
	"encoding/json"
	"fmt"
	"os"

	models "github.com/ApplePieAndCrime/go-yandex-metrics/internal/model"
)

type Producer struct {
	file    *os.File
	encoder *json.Encoder
}

func NewProducer(filename string) (*Producer, error) {
	// откройте файл и создайте для него json.Encoder
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return &Producer{
		file:    file,
		encoder: encoder,
	}, nil
}

func (p *Producer) WriteEvent(metricsList []models.Metrics) error {
	return p.encoder.Encode(metricsList)
}

func (p *Producer) Close() error {
	return p.file.Close()
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
		return nil, err
	}
	fmt.Println("metricsList: ", metricsList)
	return metricsList, nil
}

func (c *Consumer) Close() error {
	return c.file.Close()
}
