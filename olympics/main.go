//go:build !solution

package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
)

type athleteInfo struct {
	Name    string `json:"athlete"`
	Age     int    `json:"age"`
	Country string `json:"country"`
	Year    int    `json:"year"`
	Date    string `json:"date"`
	Sport   string `json:"sport"`
	Gold    int    `json:"gold"`
	Silver  int    `json:"silver"`
	Bronze  int    `json:"bronze"`
	Total   int    `json:"total"`
}

type MedalsStat struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Bronze int `json:"bronze"`
	Total  int `json:"total"`
}
type AthleteRsp struct {
	Name         string                `json:"athlete"`
	Country      string                `json:"country"`
	Medals       MedalsStat            `json:"medals"`
	MedalsByYear map[string]MedalsStat `json:"medals_by_year"`
}

type TopCountryRsp struct {
	Country string `json:"country"`
	Gold    int    `json:"gold"`
	Silver int    `json:"silver"`
	Bronze int    `json:"bronze"`
	Total   int    `json:"total"`
}

func aggrAth(sportData []athleteInfo) (athletes map[string]*AthleteRsp, topBySport map[string]map[string]*AthleteRsp,
	topByYear map[string]map[string]*TopCountryRsp) {

	athletes = make(map[string]*AthleteRsp)
	topBySport = make(map[string]map[string]*AthleteRsp)
	topByYear = make(map[string]map[string]*TopCountryRsp)
	firstCountry := make(map[string]string)

	for _, entry := range sportData {
		if _, ok := firstCountry[entry.Name]; !ok {
			firstCountry[entry.Name] = entry.Country
		}
		assignedCountry := firstCountry[entry.Name]

		rspAth, exists := athletes[entry.Name]
		if !exists {
			rspAth = &AthleteRsp{
				Name:         entry.Name,
				Country:      assignedCountry,
				MedalsByYear: make(map[string]MedalsStat),
			}
			athletes[entry.Name] = rspAth
		}

		rspAth.Medals.Gold += entry.Gold
		rspAth.Medals.Silver += entry.Silver
		rspAth.Medals.Bronze += entry.Bronze
		rspAth.Medals.Total += entry.Total

		yearStr := fmt.Sprintf("%d", entry.Year)
		ym := rspAth.MedalsByYear[yearStr]
		ym.Gold += entry.Gold
		ym.Silver += entry.Silver
		ym.Bronze += entry.Bronze
		ym.Total += entry.Total
		rspAth.MedalsByYear[yearStr] = ym

		if _, exists := topBySport[entry.Sport]; !exists {
			topBySport[entry.Sport] = make(map[string]*AthleteRsp)
		}

		sportAth, exists := topBySport[entry.Sport][entry.Name]
		if !exists {
			sportAth = &AthleteRsp{
				Name:         entry.Name,
				Country:      assignedCountry,
				MedalsByYear: make(map[string]MedalsStat),
			}
			topBySport[entry.Sport][entry.Name] = sportAth
		}

		sportAth.Medals.Gold += entry.Gold
		sportAth.Medals.Silver += entry.Silver
		sportAth.Medals.Bronze += entry.Bronze
		sportAth.Medals.Total += entry.Total

		sym := sportAth.MedalsByYear[yearStr]
		sym.Gold += entry.Gold
		sym.Silver += entry.Silver
		sym.Bronze += entry.Bronze
		sym.Total += entry.Total
		sportAth.MedalsByYear[yearStr] = sym

		if _, exists := topByYear[yearStr]; !exists {
			topByYear[yearStr] = make(map[string]*TopCountryRsp)
		}
		cr := topByYear[yearStr][entry.Country]
		if cr == nil {
			cr = &TopCountryRsp{
				Country: entry.Country,
			}
			topByYear[yearStr][entry.Country] = cr
		}
		cr.Gold += entry.Gold
		cr.Silver += entry.Silver
		cr.Bronze += entry.Bronze
		cr.Total += entry.Total
	}

	return
}

func sortedTopBySport(topBySport map[string]map[string]*AthleteRsp) map[string][]*AthleteRsp {
	sorted := make(map[string][]*AthleteRsp)

	for sport, athletes := range topBySport {
		for _, ath := range athletes {
			sorted[sport] = append(sorted[sport], ath)
		}

		slices.SortFunc(sorted[sport], func(a, b *AthleteRsp) int {
			n := cmp.Compare(b.Medals.Gold, a.Medals.Gold)
			if n != 0 {
				return n
			}
			n = cmp.Compare(b.Medals.Silver, a.Medals.Silver)
			if n != 0 {
				return n
			}
			n = cmp.Compare(b.Medals.Bronze, a.Medals.Bronze)
			if n != 0 {
				return n
			}
			return cmp.Compare(a.Name, b.Name)
		})
	}

	return sorted
}

func sortedTopByYear(topByYear map[string]map[string]*TopCountryRsp) map[string][]*TopCountryRsp {
	sorted := make(map[string][]*TopCountryRsp)

	for year, countries := range topByYear {
		for _, cr := range countries {
			sorted[year] = append(sorted[year], cr)
		}

		slices.SortFunc(sorted[year], func(a, b *TopCountryRsp) int {
			n := cmp.Compare(b.Gold, a.Gold)
			if n != 0 {
				return n
			}
			n = cmp.Compare(b.Silver, a.Silver)
			if n != 0 {
				return n
			}
			n = cmp.Compare(b.Bronze, a.Bronze)
			if n != 0 {
				return n
			}
			return cmp.Compare(a.Country, b.Country)
		})
	}

	return sorted
}

func athInfoResp(w http.ResponseWriter, r *http.Request,
	athName string, athletes map[string]*AthleteRsp) error {

	rsp, ok := athletes[athName]
	if !ok {
		http.Error(w, fmt.Sprintf("athlete %s not found", athName), http.StatusNotFound)
		return nil
	}

	rspBytes, err := json.Marshal(rsp)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(rspBytes)
	return nil
}

func topAthInSportResp(w http.ResponseWriter, r *http.Request, sport string,
	limit string, topBySport map[string][]*AthleteRsp) error {

	limitInt, err := strconv.Atoi(limit)
	if limitInt < 0 || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	athletes, ok := topBySport[sport]
	if !ok {
		http.Error(w, fmt.Sprintf("sport '%s' not found", sport), http.StatusNotFound)
		return nil
	}

	if limitInt > len(athletes) {
		limitInt = len(athletes)
	}

	rspBytes, err := json.Marshal(athletes[:limitInt])
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(rspBytes)
	return nil
}

func topCountriesInYearResp(w http.ResponseWriter, r *http.Request, year string,
	limit string, topByYear map[string][]*TopCountryRsp) error {
	result, ok := topByYear[year]
	if !ok {
		http.Error(w, fmt.Sprintf("year %s not found", year), http.StatusNotFound)
		return nil
	}

	limitInt, err := strconv.Atoi(limit)
	if limitInt < 0 || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	if limitInt > len(result) {
		limitInt = len(result)
	}

	rspBytes, err := json.Marshal(result[:limitInt])
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(rspBytes)
	return nil
}

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	dataPath := flag.String("data", "", "path to file to store data in")
	flag.Parse()

	if *dataPath == "" {
		panic("data path is required")
	}

	data, err := os.ReadFile(*dataPath)
	if err != nil {
		panic(err)
	}

	var sportData []athleteInfo
	err = json.Unmarshal(data, &sportData)
	if err != nil {
		panic(err)
	}

	athletes, topBySport, topByYear := aggrAth(sportData)

	sortedYear := sortedTopByYear(topByYear)
	sortedSp := sortedTopBySport(topBySport)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /athlete-info", func(w http.ResponseWriter, r *http.Request) {
		athName := r.URL.Query().Get("name")

		err := athInfoResp(w, r, athName, athletes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /top-athletes-in-sport", func(w http.ResponseWriter, r *http.Request) {
		defaultLimit := "3"
		params := r.URL.Query()

		if len(params) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		sport := params.Get("sport")
		limit := params.Get("limit")

		if limit == "" {
			limit = defaultLimit
		}

		err := topAthInSportResp(w, r, sport, limit, sortedSp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("GET /top-countries-in-year", func(w http.ResponseWriter, r *http.Request) {
		defaultLimit := "3"
		params := r.URL.Query()

		if len(params) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		year := params.Get("year")
		limit := params.Get("limit")

		if limit == "" {
			limit = defaultLimit
		}

		err := topCountriesInYearResp(w, r, year, limit, sortedYear)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	serv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", *port),
		Handler: mux,
	}

	err = serv.ListenAndServe()
	if err != nil {
		panic(err)
	}

}