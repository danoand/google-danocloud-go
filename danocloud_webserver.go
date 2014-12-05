package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/justinas/alice"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type yamlobject interface{}

var My_yaml_obj yamlobject

type Container struct {
	Data interface{}
}

// Function that creates a variable of type Container
func NewContainer(in_data yamlobject) Container {
	// Create the new variable
	var c Container
	c.Data = in_data

	// Return the Container
	return c
}

// Function that reads a yaml file and parses contents into a Go data structure
func readyaml(in_file string, yaml_dest *yamlobject) (int, error) {
	// Check inbound parameters
	if len(in_file) < 1 {
		// Don't have a valid file to read
		return 0, errors.New("No valid file to read")
	}

	// Read file into a buffer (NOTE: could be issues with large files!)
	yamlbuf, err := ioutil.ReadFile(in_file)
	if err != nil {
		// Error opening the file
		return 0, err
	}

	// Parse the byte slice
	err_unmarshall := yaml.Unmarshal(yamlbuf, yaml_dest)
	if err_unmarshall != nil {
		// Error decoding the yaml content
		return 0, err_unmarshall
	}

	// Return to caller
	return len(yamlbuf), nil

}

// Middleware handler function that logs inbound request information
func loggingHandler(next http.Handler) http.Handler {
	// Define the logging code
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	// Return a handler function that wraps the logging code and the core handler function
	return http.HandlerFunc(fn)
}

// Middleware handler function that recovers from a panic in the underlying request handler (if it occurs)
func recoverHandler(next http.Handler) http.Handler {
	// Define a function that defers a function to recover from a panic
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %+v", err)
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// Function that handles request to '/'
func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Declare a map to be used as the json response
	response_object := make(map[string]string)
	// Populate the map
	response_object["msg"] = "You're in the index route."

	// Marshal or create the json from the underlying map
	js, err := json.Marshal(response_object)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // If there's an error, write an error to the http response
		return
	}

	// Set the mime type of the data being returned in the response
	w.Header().Set("Content-Type", "application/json")

	// Write the json to the response
	w.Write(js)
}

// Function that handles request to '/panicme'
func panicmeHandler(w http.ResponseWriter, r *http.Request) {
	// Panic biatttch!
	panic(fmt.Sprint("Here we go... panicing keep this serva up!"))
}

// Function that handles request to '/yamlme'
func (c *Container) yamlmeHandler(w http.ResponseWriter, r *http.Request) {
	// Convert the parsed yaml object to json and write to the response
	//    marshal or create the json from the underlying object
	tmp_write_this := fmt.Sprint(c.Data)

	// Set the mime type of the data being returned in the response
	w.Header().Set("Content-Type", "text/plain")

	// Write the json to the response
	w.Write([]byte(tmp_write_this))
}

// Function that handles embedded path variables
func pathParamsHandler(w http.ResponseWriter, r *http.Request) {
	// Declare a map to be used as the json response
	response_object := make(map[string]string)
	// Populate the map
	response_object["msg"] = "You're in the /params route."

	// Parse the path information
	my_url := r.URL
	my_path := my_url.Path

	// Parse the path delimited by forward slash ('/')
	value_slice := strings.Split(my_path, "/")

	// Iterate through the values
	for i := 2; i < len(value_slice); i++ {
		my_map_key := strconv.Itoa(i)
		response_object[my_map_key] = value_slice[i]
	}

	// Marshal or create the json from the underlying map
	js, err := json.Marshal(response_object)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // If there's an error, write an error to the http response
		return
	}

	// Set the mime type of the data being returned in the response
	w.Header().Set("Content-Type", "application/json")

	// Write the json to the response
	w.Write(js)
}

// Function that serves static files/resources
func staticHandler(w http.ResponseWriter, r *http.Request) {
	// Use the http.ServeFile function
	http.ServeFile(w, r, r.URL.Path[1:]) // Use '1' to remove the leading forward slash ('/')
}

func foofile(w http.ResponseWriter, r *http.Request) {
	// Open a file for reading
	my_file, err := os.Open("go_webserver_2_file.txt")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // If there's an error, write an error to the http response
		return
	}

	// Set response headers so the browser will consider it a file download
	w.Header().Set("Content-Disposition", "attachment; filename=samplefile.txt")
	w.Header().Set("Content-Type", "text/plain")

	// Copy from the file (reading of the file) to the http response without loading contents into memory (important for big files)
	io.Copy(w, my_file)
}

// Main
func main() {
	// Read the yaml file and parse contents into my pre-declared object
	num_bytes, err := readyaml("mytest.yaml", &My_yaml_obj)
	if err != nil {
		log.Println("An error occurred executing the parsing function:", err)
		os.Exit(1)
	} else {
		log.Println("Processed a yaml file with number of bytes:", num_bytes)
	}

	// Create a data container as a store for the object that was just parsed
	my_data_store := NewContainer(My_yaml_obj)

	// Set up a middleware handler using Alice
	commonHandlers := alice.New(loggingHandler, recoverHandler)

	// Set up route handlers
	http.Handle("/params/", commonHandlers.ThenFunc(pathParamsHandler)) // IMPORTANT: Notice the '/' at the end of path pattern
	http.Handle("/panicme", commonHandlers.ThenFunc(panicmeHandler))
	http.Handle("/public/", commonHandlers.ThenFunc(staticHandler)) // IMPORTANT: Notice the '/' at the end of path pattern
  
  // Description of what's happening below...
  //   - assigning to '/yamlme' a method to a type that implements the handler interface
  //   - this enables me to "pass" data (a Struct field included in the type) that that route handler function will use
  //       (there may be other ways to acheive this)
	http.Handle("/yamlme", commonHandlers.ThenFunc(my_data_store.yamlmeHandler))
	http.Handle("/", commonHandlers.ThenFunc(indexHandler))

	// Start the web server listening on the specified port
	default_port := ":3000"
	log.Println("Starting the webserver on port:", default_port)
	http.ListenAndServe(default_port, nil)
}
