package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Courier struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TrackPoint struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	mu sync.RWMutex
	couriers = map[int]*Courier{
		1: {ID: 1, Name: "Курьер 1", Status: "free"},
		2: {ID: 2, Name: "Курьер 2", Status: "free"},
		3: {ID: 3, Name: "Курьер 3", Status: "free"},
	}
	tracks = map[int][]TrackPoint{}
	apiKey string
)

func jsonOut(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func requireKey(w http.ResponseWriter, r *http.Request) bool {
	if apiKey == "" {
		return true
	}
	key := r.Header.Get("X-API-Key")
	if key == "" {
		key = r.URL.Query().Get("key")
	}
	if key != apiKey {
		jsonOut(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return false
	}
	return true
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	jsonOut(w, 200, map[string]any{"ok": true, "service": "SALOMAT Courier Server 0.3"})
}

func listCouriers(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Courier, 0, len(couriers))
	for i := 1; i <= len(couriers); i++ {
		if c, ok := couriers[i]; ok {
			out = append(out, *c)
		}
	}
	jsonOut(w, 200, out)
}

func courierRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/couriers/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		jsonOut(w, 404, map[string]any{"ok": false})
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		jsonOut(w, 400, map[string]any{"ok": false, "error": "bad courier id"})
		return
	}
	action := parts[1]

	if action == "track" && r.Method == http.MethodGet {
		mu.RLock()
		defer mu.RUnlock()
		points := tracks[id]
		if len(points) > 500 {
			points = points[len(points)-500:]
		}
		jsonOut(w, 200, points)
		return
	}

	if !requireKey(w, r) {
		return
	}

	switch action {
	case "status":
		if r.Method != http.MethodPost {
			jsonOut(w, 405, map[string]any{"ok": false})
			return
		}
		var in struct{ Status string `json:"status"` }
		if json.NewDecoder(r.Body).Decode(&in) != nil || (in.Status != "free" && in.Status != "busy") {
			jsonOut(w, 400, map[string]any{"ok": false, "error": "bad status"})
			return
		}
		mu.Lock()
		c, ok := couriers[id]
		if ok {
			c.Status = in.Status
			c.UpdatedAt = time.Now().UTC()
		}
		mu.Unlock()
		if !ok {
			jsonOut(w, 404, map[string]any{"ok": false, "error": "courier not found"})
			return
		}
		jsonOut(w, 200, map[string]any{"ok": true})

	case "location":
		if r.Method != http.MethodPost {
			jsonOut(w, 405, map[string]any{"ok": false})
			return
		}
		var in struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil ||
			in.Latitude < -90 || in.Latitude > 90 ||
			in.Longitude < -180 || in.Longitude > 180 {
			jsonOut(w, 400, map[string]any{"ok": false, "error": "bad coordinates"})
			return
		}
		now := time.Now().UTC()
		mu.Lock()
		c, ok := couriers[id]
		if ok {
			lat, lng := in.Latitude, in.Longitude
			c.Latitude, c.Longitude, c.UpdatedAt = &lat, &lng, now
			tracks[id] = append(tracks[id], TrackPoint{Latitude: lat, Longitude: lng, CreatedAt: now})
			if len(tracks[id]) > 2000 {
				tracks[id] = tracks[id][len(tracks[id])-2000:]
			}
		}
		mu.Unlock()
		if !ok {
			jsonOut(w, 404, map[string]any{"ok": false, "error": "courier not found"})
			return
		}
		jsonOut(w, 200, map[string]any{"ok": true, "updated_at": now})

	default:
		jsonOut(w, 404, map[string]any{"ok": false})
	}
}

func dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboardHTML)
}

func main() {
	apiKey = os.Getenv("API_KEY")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", cors(health))
	http.HandleFunc("/api/couriers", cors(listCouriers))
	http.HandleFunc("/api/couriers/", cors(courierRoute))
	http.HandleFunc("/", dashboard)

	log.Printf("SALOMAT Courier Server 0.3 listening on :%s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

const dashboardHTML = `<!doctype html>
<html lang="ru"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>САЛОМАТ — Курьеры</title>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css">
<style>
:root{--g:#1e8f55;--p:#6b3fa0;--bg:#f7f8fa;--b:#e4e6ea}
*{box-sizing:border-box}body{margin:0;font-family:Arial,sans-serif;background:var(--bg)}
header{height:62px;background:#fff;border-bottom:1px solid var(--b);display:flex;align-items:center;padding:0 18px;font-weight:700}
.brand{color:var(--g);font-size:20px}.layout{display:grid;grid-template-columns:310px 1fr;gap:14px;padding:14px}
.panel{background:#fff;border:1px solid var(--b);border-radius:14px;overflow:hidden}.panel h2{font-size:17px;margin:0;padding:15px;border-bottom:1px solid var(--b)}
.stats{display:grid;grid-template-columns:1fr 1fr;gap:8px;padding:10px}.stat{background:#fafafa;border:1px solid var(--b);border-radius:11px;padding:10px}.stat b{font-size:24px;display:block}
.couriers{padding:8px}.c{border:1px solid var(--b);border-radius:11px;padding:11px;margin:7px 0;cursor:pointer}.c.active{outline:2px solid var(--p)}
.row{display:flex;justify-content:space-between;gap:8px}.name{font-weight:700}.badge{font-size:12px;padding:4px 8px;border-radius:999px}.free{background:#e5f6ed;color:#137544}.busy{background:#f2e9fb;color:#6b3fa0}.meta{font-size:12px;color:#777;margin-top:5px}
#map{height:calc(100vh - 92px);min-height:500px}.foot{padding:4px 15px 14px;color:#777;font-size:12px}
@media(max-width:760px){.layout{grid-template-columns:1fr}#map{height:60vh}}
</style></head>
<body><header><span class="brand">САЛОМАТ</span>&nbsp;— контроль курьеров 0.3</header>
<div class="layout">
<aside class="panel"><h2>Курьеры</h2><div class="stats"><div class="stat">Свободны<b id="f">0</b></div><div class="stat">Заняты<b id="b">0</b></div></div><div id="list" class="couriers"></div><div class="foot">Обновление каждые 5 секунд</div></aside>
<main class="panel"><div id="map"></div></main></div>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script>
const map=L.map('map').setView([38.5598,68.7870],13);
L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:'&copy; OpenStreetMap'}).addTo(map);
let selected=null,line=null;const markers={};
async function track(id){selected=id;drawList();const d=await(await fetch('/api/couriers/'+id+'/track')).json();if(line){map.removeLayer(line);line=null}if(d.length>1){line=L.polyline(d.map(x=>[x.latitude,x.longitude]),{color:'#6b3fa0',weight:4}).addTo(map);map.fitBounds(line.getBounds(),{padding:[30,30]})}else if(markers[id])map.setView(markers[id].getLatLng(),16)}
let latest=[];
function drawList(){const l=document.getElementById('list');l.innerHTML='';latest.forEach(c=>{const e=document.createElement('div');e.className='c'+(selected===c.id?' active':'');const t=c.updated_at&&!c.updated_at.startsWith('0001-')?new Date(c.updated_at).toLocaleTimeString():'нет сигнала';e.innerHTML='<div class="row"><span class="name">'+c.name+'</span><span class="badge '+c.status+'">'+(c.status==='busy'?'Занят':'Свободен')+'</span></div><div class="meta">Последний сигнал: '+t+'</div>';e.onclick=()=>track(c.id);l.appendChild(e)})}
async function refresh(){latest=await(await fetch('/api/couriers')).json();document.getElementById('f').textContent=latest.filter(x=>x.status==='free').length;document.getElementById('b').textContent=latest.filter(x=>x.status==='busy').length;drawList();latest.forEach(c=>{if(c.latitude==null)return;const p=[c.latitude,c.longitude],txt='<b>'+c.name+'</b><br>'+(c.status==='busy'?'Занят':'Свободен');if(!markers[c.id])markers[c.id]=L.marker(p).addTo(map).bindPopup(txt);else markers[c.id].setLatLng(p).setPopupContent(txt)})}
refresh();setInterval(refresh,5000);
</script></body></html>`
