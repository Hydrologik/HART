package main

import (
	"HART/web/mongoDrive"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"text/template"
	"web/clientTag"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var u = uint8(rand.Intn(255)) //Randomize cookies
var hconf oauth2.Config
var conf = &hconf
var IgnData map[string]interface{}

// Cookie store for persistent log-in
var (
	key   = []byte{239, 57, 183, 33, 121, 175, 214, u, 52, 235, 33, 167, 74, 91, 153, 39}
	store = sessions.NewCookieStore(key)
)

var templates = template.Must(template.ParseFiles(
	"../Templates/index.html",
	"../Templates/login.html",
	"../Templates/logout.html",
	"../Templates/ignCards.html",
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//															Handlers																	//
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func indexHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("index handler called")
	//session, _ := store.Get(r, "hydro-cookie")
	alarms, _ := mongoDrive.GetIgnAlarms(bson.D{})
	w = setHeaders(w)
	renderTemplate(w, "index", alarms)
}

func chartHandler(w http.ResponseWriter, r *http.Request) {
	alarmData, err := mongoDrive.GetIgnMetrics("All", "", "", "")
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Alarm Data"}),
		charts.WithColorsOpts(opts.Colors{"#FF7373", "#FFF68F", "#00CED1"}),
	)

	items := make([]opts.PieData, 0)
	items = append(items, opts.PieData{Name: "Alert", Value: alarmData.Alert})
	items = append(items, opts.PieData{Name: "Warn", Value: alarmData.Warn})
	items = append(items, opts.PieData{Name: "Good", Value: alarmData.Good})

	//Custom render function to remove default document setting from go-echarts
	pie.AddSeries("pie", items)
	pie.Renderer = NewSnippetRenderer(pie, pie.Validate)
	pie.Render(w)
}

func ignCardHandler(w http.ResponseWriter, r *http.Request) {
	w = setHeaders(w)
	type cardStr struct {
		Er    bool
		Ms    string
		Cs    string
		Cn    string
		Bc    []string
		Alert []clientTag.Card
		Warn  []clientTag.Card
		Good  []clientTag.Card
	}
	ty := r.URL.Query()["type"][0]
	var bcStr []string
	var data []clientTag.Card
	var err error
	var cs, cn string

	//We use switch to customize card views that will generally be very simillar
	switch ty {
	case "client":
		//Breadcrumb elements per depth
		bcStr = []string{
			"<span>Ignition</span>",
		}
		//Child String indicates the href for the next level of depth
		cs = "/ignCards?type=site&c="
		cn = "Sites"
		data, err = clientTag.GetCardList(ty, IgnData, "", "")

	case "site":
		//Breadcrumb for site + Child String
		c := r.URL.Query()["c"][0]
		bcStr = []string{
			"<a href=\"/ignCards?type=client\" hx-get=\"/ignCards?type=client\" hx-target=\"#main-body\">Ignition</a>",
			fmt.Sprintf("<span>%s</span>", c),
		}
		cs = fmt.Sprintf("/ignCards?type=tag&c=%s&s=", c)
		cn = "Tags"
		data, err = clientTag.GetCardList(ty, IgnData, c, "")
	case "tag":
		c := r.URL.Query()["c"][0]
		s := r.URL.Query()["s"][0]
		bcStr = []string{
			"<a href=\"/ignCards?type=client\" hx-get=\"/ignCards?type=client\" hx-target=\"#main-body\">Ignition</a>",
			fmt.Sprintf("<a href=\"/ignCards?type=site&c=%s\" hx-get=\"/ignCards?type=site&c=%s\" hx-target=\"#main-body\">%s</a>", c, c, c),
			fmt.Sprintf("<span>%s</span>", s),
		}
		cs = fmt.Sprintf("/ignTag?type=tag&c=%s&s=%st=", c, s)
		cn = "Tag metrics"
		data, err = clientTag.GetCardList(ty, IgnData, c, s)
	}

	//Error handling still make page just replace content with error message
	if err != nil {
		erStr := fmt.Sprintf("Error getting Card List Error %s", err.Error())
		renderTemplate(w, "ignCards", cardStr{Er: true, Ms: erStr, Bc: bcStr})
		return
	} else if len(data) == 0 {
		erStr := "Error get card returned empty list!"
		renderTemplate(w, "ignCards", cardStr{Er: true, Ms: erStr, Bc: bcStr})
		return
	}

	var alert, warn, good []clientTag.Card
	for _, c := range data {
		if c.Alert > 0 {
			alert = append(alert, c)
		} else if c.Warn > 0 {
			warn = append(warn, c)
		} else {
			good = append(good, c)
		}
	}

	renderTemplate(w, "ignCards", cardStr{Er: false, Ms: "", Bc: bcStr, Warn: warn, Alert: alert, Good: good, Cs: cs, Cn: cn})
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//															Main  																		//
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	var err error
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

	IgnData, err = clientTag.IgnCall()
	if err != nil {
		log.Fatal(err.Error())
	}

	//Special handlers for Authentication
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/validate", oauthValidate)
	http.HandleFunc("/logout", logoutHandler)

	//Standard Pages
	http.HandleFunc("/", makeHandler(indexHandler))
	http.HandleFunc("/getChart", makeHandler(chartHandler))
	http.HandleFunc("/ignCards", makeHandler(ignCardHandler))

	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("../resources"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("../js"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("../css"))))

	log.Fatal(http.ListenAndServe(":5280", nil))

}
