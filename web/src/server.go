package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var u = uint8(rand.Intn(255)) //Randomize cookies
var hconf oauth2.Config
var conf = &hconf

// Cookie store for persistent log-in
var (
	key   = []byte{239, 57, 183, 33, 121, 175, 214, u, 52, 235, 33, 167, 74, 91, 153, 39}
	store = sessions.NewCookieStore(key)
)

var templates = template.Must(template.ParseFiles(
	"../Templates/index.html",
	"../Templates/login.html",
	"../Templates/logout.html",
))

// Render the provide template string with the passed in data
func renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	//fmt.Println(data)
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Take in http resopnse writer and set re-validate headers
func setHeaders(w http.ResponseWriter) http.ResponseWriter {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.
	return w
}

// Authentication function and re-route
func authenticate(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	if auth, ok := s.Values["authenticated"].(bool); !ok || !auth {
		fmt.Println("Redirecting per auth")
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// Creates handler function that has passed authenication
func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "hydro-cookie")
		authenticate(w, r, session)
		fn(w, r)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("index handler called")
	//session, _ := store.Get(r, "hydro-cookie")
	w = setHeaders(w)
	renderTemplate(w, "index", "")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("Login handler Called")
	session, _ := store.Get(r, "hydro-cookie")
	w = setHeaders(w)

	// Redirect user to Google's consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state")

	//If already authenticated push to index
	val, ok := session.Values["authenticated"].(bool)
	if ok && val {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	renderTemplate(w, "login", url)
}

// OAuth login handler
func oauthValidate(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "hydro-cookie")
	code := r.URL.Query().Get("code")

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get user info using token
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v1/userinfo?alt=json", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	//request user data
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Decode user info JSON
	var userinfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&userinfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//Save user data and forward to index
	session.Values["email"] = userinfo["email"]
	session.Values["name"] = userinfo["given_name"]
	session.Values["authenticated"] = true
	session.Values["accessToken"] = token.AccessToken
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "hydro-cookie")
	auth, ok := session.Values["authenticated"].(bool)
	//fmt.Println("Logout Called")
	if auth && ok {
		//Google logout URL
		url := "https://accounts.google.com/o/oauth2/revoke?token=" + session.Values["accessToken"].(string)

		//logout of google
		resp, err := http.Post(url, "application/x-www-form-urlencoded", nil)
		if err != nil {
			http.NotFound(w, r)
		}
		defer resp.Body.Close()

		//app control log out clear cookie
		session.Values["authenticated"] = false
		session.Values["email"] = ""
		session.Values["name"] = ""
		session.Save(r, w)
		renderTemplate(w, "logout", "")
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	//env file for sensative data and basic Aut
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	//Google OAuth variables from env
	hconf = oauth2.Config{
		ClientID:     os.Getenv("GID"),
		ClientSecret: os.Getenv("GSC"),
		RedirectURL:  os.Getenv("RDR"),
		Scopes: []string{
			"openid",
			"email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	//Special handlers for Authentication
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/validate", oauthValidate)
	http.HandleFunc("/logout", logoutHandler)

	//Standard Pages
	http.HandleFunc("/", makeHandler(indexHandler))

	log.Fatal(http.ListenAndServe(":5280", nil))

}
