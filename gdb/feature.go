package gdb

import (
	"errors"
	"strings"
)

type GDBFeatures = map[string]GDBFeature

type GDBFeature struct {
	Enabled bool
	Value   string
}

func ParseFeatures(data string) (GDBFeatures, error) {
	features := make(GDBFeatures)

	for _, feature := range strings.Split(string(data), ";") {
		name, data, err := ParseFeature(feature)
		if err != nil {
			return nil, err
		}

		features[name] = data
	}

	return features, nil
}

func ParseFeature(data string) (string, GDBFeature, error) {
	if len(data) <= 0 {
		return "", GDBFeature{}, errors.New("empty feature")
	}

	last := data[len(data)-1]

	switch last {
	case '?':
		fallthrough
	case '+':
		return data[:len(data)-1], GDBFeature{
			Enabled: true,
		}, nil
	case '-':
		return data[:len(data)-1], GDBFeature{
			Enabled: false,
		}, nil
	default:
		sections := strings.Split(data, "=")

		if len(sections) != 2 {
			return "", GDBFeature{}, errors.New("invalid assignment")
		}

		key, value := sections[0], sections[1]

		return key, GDBFeature{
			Enabled: true,
			Value:   value,
		}, nil
	}
}
