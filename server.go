package main

import (
	"net/url"
    "strconv"
    "io/ioutil"
	"net/http"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
)

func main() {

	uc := NewLocationController(getSession())							// Getting the instance of LocationController

	r := httprouter.New()												// Create new router

	r.POST("/Locations", uc.CreateLocation)								// Adding a new address

	r.GET("/Locations/:location_id", uc.GetLocation)					// Get the location of the added address

	r.PUT("/Locations/:location_id", uc.UpdateLocation)					// Update an already existing address

	r.DELETE("/Locations/:location_id", uc.DeleteLocation )				// Delete an already existing address

	http.ListenAndServe("localhost:8080", r)							// Running the server
}

// Establish a new session
func getSession() *mgo.Session {
	s, err := mgo.Dial("mongodb://nom:root@ds045464.mongolab.com:45464/mongolocations")
	if err != nil {
		panic(err)
	}
	s.SetMode(mgo.Monotonic, true)
	return s
}

// Creating a new address
func (uc LocationController) CreateLocation(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var u InputAddress
	var oA OutputAddress
	json.NewDecoder(r.Body).Decode(&u)
	googResCoor := getGoogleCoordinates (u.Address + "+" + u.City + "+" + u.State + "+" + u.Zip);
	fmt.Println("resp is: ", googResCoor.Coordinate.Lat, googResCoor.Coordinate.Lang);
	oA.Id = bson.NewObjectId()
	oA.Name = u.Name
	oA.Address = u.Address
	oA.City= u.City
	oA.State= u.State
	oA.Zip = u.Zip
	oA.Coordinate.Lat = googResCoor.Coordinate.Lat
	oA.Coordinate.Lang = googResCoor.Coordinate.Lang
	uc.session.DB("mongolocations").C("locations").Insert(oA)
	uj, _ := json.Marshal(oA)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}

// Getting the address which is already existing
func (uc LocationController) GetLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("location_id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	oid := bson.ObjectIdHex(id)
	var o OutputAddress
	if err := uc.session.DB("mongolocations").C("locations").FindId(oid).One(&o); err != nil {
		w.WriteHeader(404)
		return
	}
	uj, _ := json.Marshal(o)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", uj)
}

// Delete an already existing address
func (uc LocationController) DeleteLocation (w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("location_id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	oid := bson.ObjectIdHex(id)
	if err := uc.session.DB("mongolocations").C("locations").RemoveId(oid); err != nil {
		w.WriteHeader(404)
		return
	}
	w.WriteHeader(200)
	fmt.Println("Address has been deleted successfully!")
}

//Update an already existing address
func (uc LocationController) UpdateLocation(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var i InputAddress
	var o OutputAddress
	id := p.ByName("location_id")
	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	oid := bson.ObjectIdHex(id)
	if err := uc.session.DB("mongolocations").C("locations").FindId(oid).One(&o); err != nil {
		w.WriteHeader(404)
		return
	}
	json.NewDecoder(r.Body).Decode(&i)
	googResCoor := getGoogleCoordinates (i.Address + "+" + i.City + "+" + i.State + "+" + i.Zip);
	fmt.Println("resp is: ", googResCoor.Coordinate.Lat, googResCoor.Coordinate.Lang);
	o.Address = i.Address
	o.City = i.City
	o.State = i.State
	o.Zip = i.Zip
	o.Coordinate.Lat = googResCoor.Coordinate.Lat
	o.Coordinate.Lang = googResCoor.Coordinate.Lang
	c := uc.session.DB("mongolocations").C("locations")
	id2 := bson.M{"_id": oid}
	err := c.Update(id2, o)
	if err != nil {
		panic(err)
	}
	uj, _ := json.Marshal(o)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}

type GoogleResponse struct {
	Results []GoogleResult
}

type GoogleResult struct {

	Address      string               `json:"formatted_address"`
	AddressParts []GoogleAddressPart  `json:"address_components"`
	Geometry     Geometry
	Types        []string
}

type GoogleAddressPart struct {

	Name      string `json:"long_name"`
	ShortName string `json:"short_name"`
	Types     []string
}

type Geometry struct {

	Bounds   Bounds
	Location Point
	Type     string
	Viewport Bounds
}
type Bounds struct {
	NorthEast, SouthWest Point
}

type Point struct {
	Lat float64
	Lng float64
}

type LocationController struct {
		session *mgo.Session
	}

type InputAddress struct {
		Name   string       `json:"name"`
		Address string 		`json:"address"`
		City string			`json:"city"`
		State string		`json:"state"`
		Zip string			`json:"zip"`
	}

type OutputAddress struct {

		Id     	bson.ObjectId 	`json:"_id" bson:"_id,omitempty"`
		Name   	string      	`json:"name"`
		Address string 			`json:"address"`
		City 	string			`json:"city" `
		State 	string			`json:"state"`
		Zip 	string			`json:"zip"`

		Coordinate struct{
			Lat string 		`json:"lat"`
			Lang string 	`json:"lang"`
		}
	}

// Reference to a LocationController with the session
func NewLocationController(s *mgo.Session) *LocationController {
	return &LocationController{s}
}

func getGoogleCoordinates (address string) OutputAddress{
	client := &http.Client{}
	reqURL := "http://maps.google.com/maps/api/geocode/json?address=" + url.QueryEscape(address) + "&sensor=false";
	req, err := http.NewRequest("GET", reqURL , nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error in sending req to google: ", err);	
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error in reading response: ", err);	
	}
	var res GoogleResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("error in unmashalling response: ", err);	
	}
	var ret OutputAddress
	ret.Coordinate.Lat = strconv.FormatFloat(res.Results[0].Geometry.Location.Lat,'f',7,64)
	ret.Coordinate.Lang = strconv.FormatFloat(res.Results[0].Geometry.Location.Lng,'f',7,64)
	return ret;
}

