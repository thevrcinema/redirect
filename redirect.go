package main

import (
	"net/http"
	"log"
	"fmt"
	"github.com/mssola/user_agent"
	"strconv"
	"regexp"
	"strings"
	"time"
)

// Result of device detection. Page if invalid or if not sure, otherwise Apple App store or Google Play store
type Result int
const (
	PAGE Result = 0 + iota
	APPSTORE
	PLAYSTORE
)

// Port to run server on, don't forget to prefix with ":"
const PORT = ":8001"

// Min IOS Version
const MIN_IOS_VERSION = 8.0

// Min Android Version
const MIN_ANDROID_VERSION = 5.0

// Check user agent string for presence of Mobile keyword
const ANDROID_STRICT = true

// Apple App store URL to redirect to
const APPLE_APP_STORE_REDIRECT_URL = "https://itunes.apple.com/us/app/appname"

// Google Play store URL to redirect to
const GOOGLE_PLAY_STORE_REDIRECT_URL = "https://play.google.com/store/apps/details?id=xxx.xxx.xxx"

// Default value for displaying debug info.
// Visit root for normal version (http://localhost:8001)
// Visit /debug for version with debug info (http://localhost:8001/debug)
var DEBUG = false

//start app
func main() {
	// declare routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/debug", debugHandler)

	// start server
	err := http.ListenAndServe(PORT, nil)

	// show errors if any
	if err != nil {
		fmt.Println("error")
		log.Fatal("ListenAndServe: ", err)
	}
}

func rootHandler(writer http.ResponseWriter, request *http.Request){
	handler(writer, request)
}
func debugHandler(writer http.ResponseWriter, request *http.Request){
	DEBUG = true
	handler(writer, request)
}

func handler(writer http.ResponseWriter, request *http.Request)  {
	// start timer
	start := time.Now()

	// Get user agent info
	ua := getUserAgent(request)

	// default result
	result := PAGE


	if DEBUG {
		//print user agent info
		showDebug(writer, ua)
	}

	//only execute if user is on a mobile device
	if isMobile(ua) == true {

		// Check iPhone
		if ua.Platform() == "iPhone" {
			version := getIphoneVersion(ua)
			if version >= MIN_IOS_VERSION {
				result = APPSTORE
				if DEBUG { fmt.Fprintf(writer, "Result: IOS version is supported. Minimum version: %f. Your version: %2f\n", MIN_IOS_VERSION, version) }
			} else {
				if DEBUG { fmt.Fprintf(writer, "Result: IOS version not supported. Needs to be at least %f. Your version is: %2f\n", MIN_IOS_VERSION, version) }
			}
		}

		// Check iPad
		if ua.Platform() == "iPad" {
			fmt.Fprint(writer, "Result: iPad is not supported\n")
		}

		// Check Android
		if isAndroid(ua) {
			version := getMobileAndroidVersion(ua)
			if version >= MIN_ANDROID_VERSION {
				result = PLAYSTORE
				if DEBUG { fmt.Fprintf(writer, "Result: Android version is supported. Minimum version: %f. Your version: %2f\n", MIN_ANDROID_VERSION, version)}
			} else {
				if DEBUG { fmt.Fprintf(writer, "Result: Android version or device not supported. Needs to be a mobile device with at least version %f. Your version is: %2f\n", MIN_IOS_VERSION, version)}
			}
		}
	}

	if DEBUG { fmt.Fprint(writer, "----------------- EOF DEBUG --------------\n\n") }

	// Work with result
	switch (result){
		// Show custom web page
		case PAGE:
		fmt.Fprint(writer, "Display custom page")

		// Redirect to app store
		case APPSTORE:
		fmt.Fprintf(writer, "Redirect to Apple App store:  %s\n", APPLE_APP_STORE_REDIRECT_URL)

		// Redirect to play store
		case PLAYSTORE:
		fmt.Fprintf(writer, "Redirect to Google Play store:  %s\n", GOOGLE_PLAY_STORE_REDIRECT_URL)
	}

	// Show duration
	fmt.Fprintf(writer, "Duration:  %v\n", time.Since(start))
}


// Debug info about user agent
func showDebug(writer http.ResponseWriter, ua *user_agent.UserAgent){
	fmt.Fprint(writer, "----------------- DEBUG INFO ---------------\n\n")
	fmt.Fprintf(writer, "Full UA?: %s\n", ua.UA())
	fmt.Fprintf(writer, "Is mobile?: %s\n", strconv.FormatBool(ua.Mobile()))
	fmt.Fprintf(writer, "Is bot?: %s\n", strconv.FormatBool(ua.Bot()))

	fmt.Fprintf(writer, "Platform: %s\n", ua.Platform())
	fmt.Fprintf(writer, "OS: %s\n", ua.OS())

	name, version := ua.Engine()
	fmt.Fprintf(writer, "Engine name: %s\n", name)
	fmt.Fprintf(writer, "Engine version: %s\n", version)

	name, version = ua.Browser()
	fmt.Fprintf(writer, "Browser name: %s\n", name)
	fmt.Fprintf(writer, "Browser version: %s\n", version)
}


// Make user agent object from http request
func getUserAgent(request *http.Request) *user_agent.UserAgent{
	return user_agent.New(request.UserAgent());
}


func getIphoneVersion(userAgent *user_agent.UserAgent) float64{
	pattern := "OS ((\\d+_?){2,3})\\s"
	return getDeviceVersion(pattern, true, userAgent)
}

func getAndroidVersion(userAgent *user_agent.UserAgent) float64{
	pattern := "Android (\\d+.\\d+)"
	return getDeviceVersion(pattern, false, userAgent)
}

func getDeviceVersion(pattern string, replace bool, userAgent *user_agent.UserAgent) float64{
	reg, _ := regexp.Compile(pattern)
	matches := reg.FindAllStringSubmatch(getOS(userAgent), 1)

	// Check if OS matches pattern and has at least 1 match
	if reg.MatchString(getOS(userAgent)) == true && len(matches) > 0{
		// get version number from matches
		version_number := matches[0][1]

		//iphone: replace _ for .
		if replace {
			version_number = strings.Replace(matches[0][1], "_", ".",-1)
		}
		// Convert to float
		f,_ := strconv.ParseFloat(string(version_number), 64)
		return f
	}
	return 0.0
}

func isAndroid(userAgent *user_agent.UserAgent) bool{
	return strings.HasPrefix(getOS(userAgent), "Android")
}

func getMobileAndroidVersion(userAgent *user_agent.UserAgent) float64{

	// If ANDROID_STRICT is true  check if Chromium or Mozilla browser is a mobile browser.
	if(ANDROID_STRICT) {

		// The pattern for a mobile browser
		pattern := "Mobile Safari/{1}((\\d+.){2,3})"
		reg, _ := regexp.Compile(pattern)

		// If it is no valid mobile browser
		if !reg.MatchString(userAgent.UA()) && userAgent.OS() != "Mobile"{
			return 0.0
		}
	}

	// return version number. If invalid it return 0.0
	version := getAndroidVersion(userAgent)
	return version
}

func getOS(userAgent *user_agent.UserAgent) string{
	// When on a Mozilla browser, the OS is "Mobile" and the (Android) device info is available the platform property.
	if(userAgent.OS() == "Mobile"){
		return  userAgent.Platform()
	}
	// Other devices/browsers
	return userAgent.OS()
}

func isMobile(userAgent *user_agent.UserAgent) bool{
	// When on a Mozilla browser, the Mobile() method returns false, but the OS is Mobile.
	return userAgent.Mobile() || userAgent.OS() == "Mobile"
}