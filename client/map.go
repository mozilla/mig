// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package client /* import "mig.ninja/mig/client" */

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
)

type CommandLocation struct {
	Endpoint      string   `json:"endpoint"`
	CommandID     float64  `json:"commandid"`
	ActionID      float64  `json:"actionid"`
	FoundAnything bool     `json:"foundanything"`
	ConnectionsTo []string `json:"connections_to"`
	Latitude      float64  `json:"latitude"`
	Longitude     float64  `json:"longitude"`
	City          string   `json:"city"`
	Country       string   `json:"country"`
}

func ValueToLocation(v interface{}) (cl CommandLocation, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToLocation) -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &cl)
	if err != nil {
		panic(err)
	}
	return
}

func PrintMap(locs []CommandLocation, title string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PrintMap() -> %v", e)
		}
	}()
	gmap := makeMapHeader(title)
	locs = singularizeLocations(locs)
	data, err := json.Marshal(locs)
	if err != nil {
		panic(err)
	}
	gmap += fmt.Sprintf(`        <script type="text/javascript"> var endpoints = %s </script>`, data)
	var details []string
	details = append(details, "        <ol>\n")
	for _, cl := range locs {
		detail := fmt.Sprintf("            <li>%s: found=%t</li>", cl.Endpoint, cl.FoundAnything)
		details = append(details, detail)
	}
	details = append(details, "        </ol>\n")
	gmap += makeMapFooter(title, details)

	// write map data to temp file
	fd, err := ioutil.TempFile("", "migmap_")
	defer fd.Close()
	if err != nil {
		panic(err)
	}
	_, err = fd.Write([]byte(gmap))
	if err != nil {
		panic(err)
	}
	fi, err := fd.Stat()
	if err != nil {
		panic(err)
	}
	filepath := fmt.Sprintf("%s/%s", os.TempDir(), fi.Name())
	fmt.Fprintf(os.Stderr, "map written to %s\n", filepath)
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("firefox", filepath).Start()
	case "darwin":
		err = exec.Command("open", "-b", "org.mozilla.firefox", filepath).Start()
	default:
		return
	}
	if err != nil {
		panic(err)
	}
	return
}

// singularizeLocations prevent multiple point from using the same coordinates, and therefore show as one point on the map
func singularizeLocations(orig_locs []CommandLocation) (locs []CommandLocation) {
	locs = orig_locs
	for i, _ := range locs {
		for j := 0; j < i; j++ {
			if locs[i].Latitude == locs[j].Latitude && locs[i].Longitude == locs[j].Longitude {
				switch i % 8 {
				case 0:
					locs[i].Latitude += 0.005
				case 1:
					locs[i].Longitude += 0.005
				case 2:
					locs[i].Latitude -= 0.005
				case 3:
					locs[i].Longitude -= 0.005
				case 4:
					locs[i].Latitude += 0.005
					locs[i].Longitude += 0.005
				case 5:
					locs[i].Latitude -= 0.005
					locs[i].Longitude -= 0.005
				case 6:
					locs[i].Latitude += 0.005
					locs[i].Longitude -= 0.005
				case 7:
					locs[i].Latitude -= 0.005
					locs[i].Longitude += 0.005
				}
			}
		}
	}
	return
}

func makeMapHeader(title string) string {
	return fmt.Sprintf(`
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
    <head>
        <meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
        <title>%s</title>
        <script type="text/javascript" src="https://maps.googleapis.com/maps/api/js?v=3.exp&amp;signed_in=true"></script>
        <script type="text/javascript" src="https://raw.githubusercontent.com/googlemaps/js-marker-clusterer/gh-pages/src/markerclusterer_compiled.js"></script>
`, title)
}
func makeMapFooter(title string, body []string) (footer string) {
	footer = `
        <script type="text/javascript">
var locs = new Array();
var marker = new Array();
var connections = new Array();
var cluster = new Array();
var arrowSymbol = {
    path: google.maps.SymbolPath.CIRCLE,
    scale: 2,
    strokeColor: 'blue'
};
function initialize() {
    var center = new google.maps.LatLng(0,0);
    var mapOptions = {
        zoom: 2,
        center: center,
        mapTypeId: google.maps.MapTypeId.TERRAIN
    };
    var map = new google.maps.Map(
        document.getElementById('map'),
        mapOptions
    );
    endpointscount = endpoints.length;
    for (var i=0; i<endpointscount; i++) {
        locs[endpoints[i].endpoint] = new google.maps.LatLng(endpoints[i].latitude, endpoints[i].longitude);
        marker[endpoints[i].endpoint] = new google.maps.Marker({
            position: locs[endpoints[i].endpoint],
            map: map,
            title: endpoints[i].endpoint
        });
        cluster.push(marker[endpoints[i].endpoint]);
    }
    for (var i=0; i<endpointscount; i++) {
        if ( endpoints[i].connections_to == null ) {
            continue
        }
        for (var j=0; j<endpoints[i].connections_to.length; j++) {
            connections[i+j] = new google.maps.Polyline({
                path: [locs[endpoints[i].endpoint], locs[endpoints[i].connections_to[j]]],
                geodesic: true,
                strokeColor: 'blue',
                strokeOpacity: 1.0,
                strokeWeight: 1,
                icons: [{
                    icon: arrowSymbol,
                    offset: '100%'
                }],
                map: map
            });
        }
    }
    animateCircle();
    var markerCluster = new MarkerClusterer(map, cluster);
}
// Use the DOM setInterval() function to change the offset of the symbol
// at fixed intervals.
function animateCircle() {
    var count = 0;
    window.setInterval(function() {
        count = (count + 1) % 200;
        conncount = connections.length;
        for (var i=0; i<conncount; i++) {
            var icons = connections[i].get('icons');
            icons[0].offset = (count / 2) + '%';
            connections[i].set('icons', icons);
        }
    }, 10);
}

google.maps.event.addDomListener(window, 'load', initialize);
        </script>
        <style type="text/css">
            #map {
                width:100%;
                height:600px;
            }
        </style>
    </head>
    <body>
`
	footer += fmt.Sprintf("        <p><b>%s</b></p>\n", title)
	footer += `<div id="map"></div>`
	for _, p := range body {
		footer += p + "\n"
	}
	footer += `
    </body>
</html>`
	return
}
