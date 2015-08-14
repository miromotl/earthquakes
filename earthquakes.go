/// List of Earthquakes
/// Queries USGS json service for a list of earthquakes and returns
/// a list with watered down information.
///
/// The list can be filtered by country code.
/// For filtering by country, the latitude/longitude of the earthquake is used
/// to get the country code from api.geonames.org
///
/// Example
/// latitude: -116.6920013, longitude: 33.5480003
/// http://api.geonames.org/countryCode?lat=33.54&lng=-116.69&username=demo ==> US

package main

import (
    "fmt"
    "log"
	"net/http"
    //"strings"
	"encoding/json"
)

// Constants with html code for our web page
const (
    pageTop = `
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <title>earthquakes</title>
            <meta charset="utf-8">
            <meta name="viewport" content="width=device-width, initial-scale=1">
            <!-- Latest compiled and minified CSS -->
            <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap.min.css">
            <!-- Latest compiled and minified JavaScript -->
            <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/js/bootstrap.min.js"></script>
        </head>
        <body>
            <div class="container">
                <h2>Earthquakes</h2>
                <p>Shows latest earthquakes around the world</p>`
    form = `
                <form role="form" action="/" method="POST">
                	<h3>Choose time span</h3>
                    <div class="radio">
                        <label><input type="radio" name="opttime" id="timeHour" value="hour">Past Hour</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="opttime" id="timeDay" value="day" checked="checked">Past Day</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="opttime" id="timeWeek" value="week">Past 7 Days</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="opttime" id="timeMonth" value="month">Past 30 Days</label>
                    </div>
                    <h3>Choose magnitude</h3>
                    <div class="radio">
                        <label><input type="radio" name="optmagnitude" id="magnitudeSignificant" value="significant" checked="checked">Significant</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="optmagnitude" id="magnitude4_5" value="4_5">M4.5+</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="optmagnitude" id="magnitude2_5" value="2_5">M2.5+</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="optmagnitude" id="magnitude1_0" value="1_0">M1.0+</label>
                    </div>
                    <div class="radio">
                        <label><input type="radio" name="optmagnitude" id="magnitudeAll" value="all">All</label>
                    </div>
                    <button type="submit" class="btn btn-success" >Show</button>
                </form>`
    pageBottom = `
            </div>
        </body>
        </html>`
    anError = `<br /><p class="text-danger">%s</p>`
)

// Enums for time spans
type timespan int;
const (
	hour timespan = iota
	day
	week
	month
)

// Enums for magnitudes
type magnitude int;
const (
	significant magnitude = iota
	m4_5
	m2_5
	m1_0
	all
)

// Struct holding the user's options and list of earthquakes
type earthquakes struct {
    ts timespan
    mag magnitude
    title string
    count int
}

func main() {
    // Setup the web server handling the requests
    http.HandleFunc("/", homePage)
    if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
        log.Fatal("failed to start server", err)
    }
}

// Handling the call to the home page; i.e. handling everything because there
// is no other page!
func homePage(writer http.ResponseWriter, request *http.Request) {
    err := request.ParseForm() // Must be called before writing the response
    fmt.Fprint(writer, pageTop, form)
    
    if err != nil {
        fmt.Fprintf(writer, anError, err)
    } else {
        if ts, mag, msg, ok := processRequest(request); ok {
            //fmt.Fprint(writer, "<p>timespan: ", ts, "</p>")
            //fmt.Fprint(writer, "<p>magnitude: ", mag, "</p>")
            if quakes, err := getQuakes(ts, mag); err != nil {
                fmt.Fprintf(writer, anError, err.Error())
            } else {
                fmt.Fprint(writer, formatQuakes(quakes))
            }
        } else if msg != "" {
            fmt.Fprintf(writer, anError, msg)
        }
    }
    
    fmt.Fprint(writer, pageBottom)
}

// Process the http request
func processRequest(request *http.Request) (timespan, magnitude, string, bool) {

    inputTs := request.Form.Get("opttime"); 
    inputMag := request.Form.Get("optmagnitude");
    
	log.Print("ts: ", inputTs)
	log.Print("mag: ", inputMag)
	
	var ts timespan
	var mag magnitude
	
	switch inputTs {
	    case "hour":
	        ts = hour
	    case "day":
	        ts = day
	    case "week":
	        ts = week
	    case "month":
	        ts = month
	    default:
	        var msg string
	        if inputTs != "" {
	            msg = "invalid timespan " + "'" + inputTs + "'"
	        }
            return day, significant, msg, false
	}
	
	switch inputMag {
	    case "significant":
	        mag = significant
	    case "4_5":
	        mag = m4_5
	    case "2_5":
	        mag = m2_5
	    case "1_0":
	        mag = m1_0
	    case "all":
	        mag = all
	    default:
	        var msg string
	        if inputMag != "" {
	            msg = "invalid magnitude " + "'" + inputMag + "'"
	        }
	        return day, significant, msg, false
	}

	return ts, mag, "", true
}

// get the earthquakes from the USGS website
func getQuakes(ts timespan, mag magnitude) (earthquakes, error) {
    url := "http://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/"
    
    switch mag {
        case significant:
            url += "significant"
        case m4_5:
            url += "4.5"
        case m2_5:
            url += "2.5"
        case m1_0:
            url += "1.0"
        case all:
            url += "all"
        default:
            url += ""
    }
    
    switch ts {
        case hour:
        url += "_hour"
        case day:
        url += "_day"
        case week:
        url += "_week"
        case month:
        url += "_month"
        default:
        url += ""
    }
    
    url += ".geojson"
    
    resp, err := http.Get(url)
    if err != nil {
        return earthquakes{}, err
    }
    
    defer resp.Body.Close()
    
    var d geojson
    
    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return earthquakes{}, err
    }
	log.Print("title: ", d.Metadata.Title)
    
    var quakes earthquakes
    quakes.ts = ts
    quakes.mag = mag
    quakes.title = d.Metadata.Title
    quakes.count = d.Metadata.Count
    
    return quakes, nil
}

// format earthquakes in HTML
func formatQuakes(quakes earthquakes) string {
    return fmt.Sprintf(`
        <h3>%s</h3>
        <p>count: %d</p>`, quakes.title, quakes.count)
}

// The GeoJSON struct
type geojson struct {
    Metadata struct {
        Url   string `json:"url"`
        Title string `json:"title"`
        Count int `json:"count"`
    } `json:"metadata"`
}