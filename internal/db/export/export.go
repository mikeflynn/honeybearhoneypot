package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mikeflynn/honeybearhoneypot/internal/db"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

type ExportFormat string
type DataType string

const (
	JSON ExportFormat = "json"
	CSV  ExportFormat = "csv"
	RAW  ExportFormat = "raw"

	DataTypeEvents  DataType = "events"
	DataTypeOptions DataType = "options"
	DataTypeCTFs    DataType = "ctf"
)

func ExportDatabase(dataTypes []DataType, outputFormat ExportFormat, outputPath string) error {
	// Check if outputPath is a directory
	info, err := os.Stat(outputPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("outputPath does not exist")
	}

	if !info.IsDir() {
		return fmt.Errorf("outputPath must be a directory")
	}

	switch outputFormat {
	case JSON:
		if len(dataTypes) == 0 {
			dataTypes = []DataType{DataTypeEvents, DataTypeOptions, DataTypeCTFs}
		}
		return jsonExport(dataTypes, outputPath)
	case CSV:
		if len(dataTypes) == 0 {
			dataTypes = []DataType{DataTypeEvents, DataTypeOptions, DataTypeCTFs}
		}
		return csvExport(dataTypes, outputPath)
	case RAW:
		if len(dataTypes) != 0 {
			return fmt.Errorf("raw export does not support specific data types")
		}

		return rawExport(outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", outputFormat)
	}
}

// jsonExport exports the specified data types to JSON files in the outputPath directory as separate files.
func jsonExport(dataTypes []DataType, outputPath string) error {
	for _, dt := range dataTypes {
		var filename string
		var data any
		var err error

		switch dt {
		case DataTypeEvents:
			filename = "events.json"
			data, err = entity.EventQuery("SELECT * FROM events")
		case DataTypeOptions:
			filename = "options.json"
			data, err = entity.OptionsAll()
		case DataTypeCTFs:
			filename = "ctf.json"
			data, err = entity.CTFUsersAll()
		default:
			continue
		}

		if err != nil {
			return err
		}

		filePath := filepath.Join(outputPath, filename)
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			return err
		}
	}
	return nil
}

// csvExport exports the specified data types to CSV files in the outputPath directory as separate files.
func csvExport(dataTypes []DataType, outputPath string) error {
	for _, dt := range dataTypes {
		var filename string
		var header []string
		var rows [][]string
		var err error

		switch dt {
		case DataTypeEvents:
			filename = "events.csv"
			header = []string{"id", "user", "host", "app", "source", "type", "action", "timestamp"}
			events, e := entity.EventQuery("SELECT * FROM events")
			err = e
			if err == nil {
				for _, evt := range events {
					rows = append(rows, []string{
						strconv.Itoa(evt.ID), evt.User, evt.Host, evt.App, evt.Source, evt.Type, evt.Action, evt.Timestamp.Format(time.RFC3339),
					})
				}
			}
		case DataTypeOptions:
			filename = "options.csv"
			header = []string{"name", "value", "timestamp"}
			options, e := entity.OptionsAll()
			err = e
			if err == nil {
				for _, opt := range options {
					rows = append(rows, []string{
						opt.Name, opt.Value, opt.Timestamp.Format(time.RFC3339),
					})
				}
			}
		case DataTypeCTFs:
			filename = "ctf.csv"
			header = []string{"id", "username", "points", "created_at"}
			users, e := entity.CTFUsersAll()
			err = e
			if err == nil {
				for _, u := range users {
					rows = append(rows, []string{
						strconv.Itoa(u.ID), u.Username, strconv.Itoa(u.Points), u.CreatedAt.Format(time.RFC3339),
					})
				}
			}
		default:
			continue
		}

		if err != nil {
			return err
		}

		filePath := filepath.Join(outputPath, filename)
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		if err := writer.Write(header); err != nil {
			return err
		}
		if err := writer.WriteAll(rows); err != nil {
			return err
		}
		writer.Flush()
	}
	return nil
}

// rawExport copies the raw sqlite database file to the outputPath directory.
func rawExport(outputPath string) error {
	srcPath := db.GetDBPath()
	if srcPath == "" {
		return fmt.Errorf("database path not found")
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstPath := filepath.Join(outputPath, "database.db")
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
