package utils

import (
	"encoding/json"
	"errors"
	"net/http"
)

const PINS_URL = "http://pinning.solenopsys.org/select"

type PinningMapping struct {
	librariesMapping map[string]string
	findKey          string
	findValue        string
	valueExtractKey  string
	byName           bool
}

func (pm *PinningMapping) LoadMapping(value string) error {
	var selectType = "pins"
	if pm.byName {
		selectType = "names"
	}
	url := PINS_URL + "/" + selectType + "?name=" + pm.findKey + "&value=" + value
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jsonDecoded map[string]map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&jsonDecoded); err != nil {
		return err
	}

	for key, value := range jsonDecoded {
		v := value[pm.valueExtractKey]
		pm.librariesMapping[v] = key
	}

	return nil
}

func (pm *PinningMapping) GetCid(libName string) (string, error) {
	EMPTY := ""
	libCid := pm.librariesMapping[libName]
	if libCid == "" {
		err := pm.LoadMapping(libName)
		if err != nil {
			return EMPTY, err
		}
		libCid = pm.librariesMapping[libName]
		if libCid == "" {
			return EMPTY, errors.New("library not found")
		}
	}
	return libCid, nil
}

func NewPinningMapping(key string, value string, valueExtractKey string, byName bool) *PinningMapping {
	return &PinningMapping{
		librariesMapping: make(map[string]string),
		findKey:          key,
		findValue:        value,
		valueExtractKey:  valueExtractKey,
		byName:           byName,
	}
}
