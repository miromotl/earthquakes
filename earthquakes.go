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
    "time"
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
                    <div class="row" id="row2">
                    <div class="col-xs-3 col">
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
                    </div>
                    <div class="col-xs-3 col">
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
                    </div>
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

func (ts timespan) String() string {
    switch ts {
        case hour:
            return "hour"
        case day:
            return "day"
        case week:
            return "week"
        case month:
            return "month"
        default:
            return ""
    }
}

func (_ timespan) Create(ts string) (timespan, error) {
	switch ts {
	    case "hour":
	        return hour, nil
	    case "day":
	        return day, nil
	    case "week":
	        return week, nil
	    case "month":
	        return month, nil
	    case "":
	        return day, nil
	    default:
            return day, fmt.Errorf("invalid timespan '%s'", ts)
	}
}

// Enums for magnitudes
type magnitude int;
const (
	significant magnitude = iota
	m4_5
	m2_5
	m1_0
	all
)

func (mag magnitude) String() string {
    switch mag {
        case significant:
            return "significant"
        case m4_5:
            return "4.5"
        case m2_5:
            return "2.5"
        case m1_0:
            return "1.0"
        case all:
            return "all"
        default:
            return ""
    }
}

func (_ magnitude) Create(mag string) (magnitude, error) {
	switch mag {
	    case "significant":
	        return significant, nil
	    case "4_5":
	        return m4_5, nil
	    case "2_5":
	        return m2_5, nil
	    case "1_0":
	        return m1_0, nil
	    case "all":
	        return all, nil
	    case "":
	        return significant, nil
	    default:
            return significant, fmt.Errorf("invalid magnitude '%s'", mag)
	}
}

// Struct holding the user's options and list of earthquakes
type earthquakes struct {
    ts timespan
    mag magnitude
    title string
    count int
    quakes []earthquake
}

// Struct holding information for one earthquake
type earthquake struct {
    mag float32
    place string
    time string
    url string
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
	var err error
	
	if ts, err = ts.Create(inputTs); err != nil {
	    return day, significant, fmt.Sprint(err), false
	}
	    
	if mag, err = mag.Create(inputMag); err != nil {
	    return day, significant, fmt.Sprint(err), false
	}

	return ts, mag, "", true
}

// get the earthquakes from the USGS website
func getQuakes(ts timespan, mag magnitude) (earthquakes, error) {
    url := "http://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/"
    url += mag.String() + "_" + ts.String() + ".geojson"

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
    quakes.quakes = make([]earthquake, d.Metadata.Count)
    
    for i, q := range d.Features {
        quakes.quakes[i].mag = q.Properties.Mag
        quakes.quakes[i].place = q.Properties.Place
        quakes.quakes[i].time = fmt.Sprint(time.Unix(q.Properties.Time/1000, 0))
        quakes.quakes[i].url = q.Properties.Url
    }
    
    return quakes, nil
}

// format earthquakes in HTML
func formatQuakes(quakes earthquakes) string {
    quakesHtml := fmt.Sprintf(`
        <h3>%s</h3>
        <p>count: %d</p>
        <div class="table-responsive">
            <table class="table">
                <thead>
                    <tr><th>Magnitude</th><th>Place</th><th>Time</th><th>Link</th></tr>
                </thead>
                <tbody>`,
                quakes.title, quakes.count)
                
    for _, q := range quakes.quakes {
        quakesHtml += fmt.Sprintf(`
                    <tr><td>%.2f</td><td>%s</td><td>%s</td><td><a href="%s">%s</a></td></tr>`,
                    q.mag, q.place, q.time, q.url, q.url)
    }
    
    quakesHtml += fmt.Sprintf(`
                </tbody>
            </table>
        </div>`)
        
    return quakesHtml
}

// The GeoJSON struct with the fields we are interested in
type geojson struct {
    Metadata struct {
        Url   string `json:"url"`
        Title string `json:"title"`
        Count int `json:"count"`
    } `json:"metadata"`
    Features [] struct {
        Properties struct {
            Mag float32 `json:"mag"`
            Place string `json:"place"`
            Time int64 `json:"time"`
            Tz int64 `json:"tz"`
            Url string `json:"url"`
        } `json:"properties"`
    } `json:"features"`
}