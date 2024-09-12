package maxmind

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/oschwald/maxminddb-golang"
)

type GeoIP struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func GetCountry(c echo.Context) string {
	ip := net.ParseIP(c.RealIP())
	db, err := maxminddb.Open("/usr/share/GeoIP/GeoLite2-Country.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var record GeoIP
	if err := db.Lookup(ip, &record); err != nil {
		log.Fatal(err)
	}

	return strings.ToLower(record.Country.ISOCode)
}

func GetCountryHandler(c echo.Context) error {
	country := GetCountry(c)
	return c.String(http.StatusOK, fmt.Sprintf("Country: %s", country))
}

func GetCountryHandlerJSON(c echo.Context) error {
	country := GetCountry(c)
	return c.JSON(http.StatusOK, map[string]string{"country": country})
}
