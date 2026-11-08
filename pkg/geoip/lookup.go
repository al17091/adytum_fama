package geoip

import (
	"fmt"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// GeoInfo holds resolved geographic and autonomous system metadata for an IP
type GeoInfo struct {
	CountryCode string
	CountryName string
	CityName    string
	Latitude    float32
	Longitude   float32
	ASN         uint32
	ASOrg       string
}

// Resolver handles memory-mapped lookups against GeoLite2 MMDB databases
type Resolver struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
	mu     sync.RWMutex
}

// NewResolver initializes City and ASN MMDB readers from local paths
func NewResolver(cityPath, asnPath string) (*Resolver, error) {
	r := &Resolver{}

	if cityPath != "" {
		cityReader, err := geoip2.Open(cityPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open City database (%s): %w", cityPath, err)
		}
		r.cityDB = cityReader
	}

	if asnPath != "" {
		asnReader, err := geoip2.Open(asnPath)
		if err != nil {
			if r.cityDB != nil {
				_ = r.cityDB.Close()
			}
			return nil, fmt.Errorf("failed to open ASN database (%s): %w", asnPath, err)
		}
		r.asnDB = asnReader
	}

	return r, nil
}

// Close releases the memory-mapped MMDB files
func (r *Resolver) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cityDB != nil {
		_ = r.cityDB.Close()
		r.cityDB = nil
	}
	if r.asnDB != nil {
		_ = r.asnDB.Close()
		r.asnDB = nil
	}
}

// Lookup resolves geolocation and ASN for a given IP address string
func (r *Resolver) Lookup(ipStr string) GeoInfo {
	info := GeoInfo{
		CountryCode: "XX",
		CountryName: "Unknown",
		CityName:    "Unknown",
		ASOrg:       "Unknown",
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return info
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Resolve City & Coordinates
	if r.cityDB != nil {
		if record, err := r.cityDB.City(ip); err == nil && record != nil {
			if record.Country.IsoCode != "" {
				info.CountryCode = record.Country.IsoCode
			}
			if name, ok := record.Country.Names["en"]; ok && name != "" {
				info.CountryName = name
			}
			if cityName, ok := record.City.Names["en"]; ok && cityName != "" {
				info.CityName = cityName
			}
			info.Latitude = float32(record.Location.Latitude)
			info.Longitude = float32(record.Location.Longitude)
		}
	}

	// 2. Resolve ASN & Organization
	if r.asnDB != nil {
		if asnRecord, err := r.asnDB.ASN(ip); err == nil && asnRecord != nil {
			info.ASN = uint32(asnRecord.AutonomousSystemNumber)
			if asnRecord.AutonomousSystemOrganization != "" {
				info.ASOrg = asnRecord.AutonomousSystemOrganization
			}
		}
	}

	return info
}
