package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mutex   sync.Mutex
	regexen = make(map[string]*regexp.Regexp)
	relock  sync.Mutex
)

type mongo_user struct {
	Fieldname  string `json: "Field string"`
	Fieldemail string `json: "Field string"`
	Fieldid    string `json: "Field string"`
	Fieldpass  []byte `json: "Field byte"`
}
type user struct {
	Name  string `json: "Name"`
	Email string `json: "Email"`
	Id    string `json: "Id"`
	Pass  string `json: "Pass"`
}
type find struct {
	Fieldid string `json: "Field string"`
}

type mongo_post struct {
	Fielduserid  string `json: "Field string"`
	Fieldid      string `json: "Field string"`
	Fieldcaption string `json: "Field string"`
	Fieldurl     string `json: "Field string"`
	Fieldtmstp   string `json: "Field string"`
}
type post struct {
	Userid  string `json: "Userid"`
	Id      string `json: "Id"`
	Caption string `json: "Caption"`
	Url     string `json: "Url"`
	Tmstp   string `json: "Tmstp"`
}

//function to check error
func CheckError(err error) {
	if err != nil {
		mutex.Unlock()
		panic(err)
	}
}

//function to connect to databse
func connect() (*mongo.Client, context.Context, context.CancelFunc) {
	mutex.Lock()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	fmt.Println("ClientOptopm Type:", reflect.TypeOf(clientOptions), "\n")

	client, err := mongo.Connect(context.TODO(), clientOptions)
	CheckError(err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	return client, ctx, cancel
}

//function to close connection to database
func close(client *mongo.Client, ctx context.Context, cancel context.CancelFunc) {

	defer cancel()
	defer func() {
		err := client.Disconnect(ctx)
		CheckError(err)
	}()
	mutex.Unlock()
}

//function to create 32bit key for encrypt function
func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

//function to encrypt password
func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	CheckError(err)
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

//function to create new user
func create_user(name, email, id string, pass []byte) {

	client, ctx, cancel := connect()
	col := client.Database("Insta").Collection("User")
	one := mongo_user{
		Fieldname:  name,
		Fieldemail: email,
		Fieldid:    id,
		Fieldpass:  pass,
	}
	result1, err := col.InsertOne(ctx, one)
	CheckError(err)
	newID := result1.InsertedID
	fmt.Println("User Created ", newID)
	close(client, ctx, cancel)
}

//function to create new post
func create_post(userid, id, caption, url string) {

	client, ctx, cancel := connect()
	col := client.Database("Insta").Collection("Post")
	one := mongo_post{
		Fielduserid:  userid,
		Fieldid:      id,
		Fieldcaption: caption,
		Fieldurl:     url,
		Fieldtmstp:   time.Now().String(),
	}
	result1, err := col.InsertOne(ctx, one)
	CheckError(err)
	newID := result1.InsertedID
	fmt.Println("Post Created ", newID)
	close(client, ctx, cancel)
}

//function to find user details using user id
func find_user(p string) mongo_user {
	client, ctx, cancel := connect()
	col := client.Database("Insta").Collection("User")
	var result mongo_user
	filter := find{
		Fieldid: p,
	}
	err := col.FindOne(ctx, filter).Decode(&result)
	CheckError(err)
	fmt.Println("User Found ", result.Fieldid)
	close(client, ctx, cancel)
	return result
}

//function to find post details using post id
func find_post(p string) mongo_post {
	client, ctx, cancel := connect()
	col := client.Database("Insta").Collection("Post")
	var result mongo_post
	filter := find{
		Fieldid: p,
	}
	err := col.FindOne(ctx, filter).Decode(&result)
	CheckError(err)
	fmt.Println("Post Found ", result.Fieldid)
	close(client, ctx, cancel)
	return result
}

//function to find all posts of a given user id
func find_all_post(p string) ([]mongo_post, int) {
	client, ctx, cancel := connect()
	col := client.Database("Insta").Collection("Post")

	var filter struct {
		Fielduserid string
	}
	filter.Fielduserid = p
	cur, err := col.Find(ctx, filter)
	CheckError(err)
	defer cur.Close(ctx)
	var arr []mongo_post
	var i int = 0
	for cur.Next(ctx) {
		var t mongo_post
		er := cur.Decode(&t)
		CheckError(er)
		arr = append(arr, t)
		i++
	}
	fmt.Println("Collection type: ", reflect.TypeOf(col), "\n")
	fmt.Println(arr)
	close(client, ctx, cancel)
	return arr, i
}

func apiResponse(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	p := r.URL.Path
	switch {
	case match(p, "/users/posts/[0-9a-zA-Z]{1,}") && r.Method == "GET":
		q := r.URL.Query().Get("page")
		if q == "" {
			q = "1"
		}
		usr_id := p[13:]
		page_no, err := strconv.Atoi(q)
		CheckError(err)
		fmt.Println(usr_id)
		fmt.Println(q)
		res, cnt := find_all_post(usr_id)
		start := ((page_no - 1) * 20)
		end := int(math.Min(float64(((page_no) * 20)), float64(cnt)))
		if end <= start {
			w.Write([]byte(`{"message": "Not_Enough_Pages"}`))
		} else {
			detail, _ := json.Marshal(res[start:end])
			fmt.Println(cnt)
			w.Write(detail)
		}
		break

	case match(p, "/users/[0-9a-zA-Z]{1,}") && r.Method == "GET":
		usr_id := p[7:]
		fmt.Println(usr_id)
		res := find_user(usr_id)
		user, _ := json.Marshal(res)
		w.Write(user)
		break

	case match(p, "/posts/[0-9a-zA-Z]{1,}") && r.Method == "GET":
		post_id := p[7:]
		fmt.Println(post_id)
		res := find_post(post_id)
		post, _ := json.Marshal(res)
		w.Write(post)
		break

	case match(p, "/users") && r.Method == "POST":
		body, err := ioutil.ReadAll(r.Body)
		CheckError(err)
		var t user
		err = json.Unmarshal(body, &t)
		CheckError(err)
		key := "any_random_key"
		pass_encrypt := encrypt([]byte(key), t.Pass)
		create_user(t.Name, t.Email, t.Id, pass_encrypt)
		w.Write([]byte(`{"message": "User Created Successfully"}`))
		break

	case match(p, "/posts") && r.Method == "POST":
		body, err := ioutil.ReadAll(r.Body)
		CheckError(err)
		var t post
		err = json.Unmarshal(body, &t)
		CheckError(err)
		create_post(t.Userid, t.Id, t.Caption, t.Url)
		w.Write([]byte(`{"message": "Post Created Successfully"}`))
		break

	default:
		http.NotFound(w, r)
		return
	}
}

func mustCompileCached(pattern string) *regexp.Regexp {
	relock.Lock()
	defer relock.Unlock()

	regex := regexen[pattern]
	if regex == nil {
		regex = regexp.MustCompile("^" + pattern + "$")
		regexen[pattern] = regex
	}
	return regex
}

func match(path, pattern string, vars ...interface{}) bool {
	regex := mustCompileCached(pattern)
	matches := regex.FindStringSubmatch(path)
	if len(matches) <= 0 {
		return false
	}
	for i, match := range matches[1:] {
		switch p := vars[i].(type) {
		case *string:
			*p = match
		case *int:
			n, err := strconv.Atoi(match)
			if err != nil {
				return false
			}
			*p = n
		default:
			panic("vars must be *string or *int")
		}
	}
	return true
}

func main() {

	http.HandleFunc("/", apiResponse)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
