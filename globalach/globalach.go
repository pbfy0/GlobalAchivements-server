package globalach
import (
    "fmt"
    "net/http"
    "appengine"
    "appengine/urlfetch"
    "appengine/datastore"
    "io/ioutil"
    "encoding/json"
//    "reflect"
)

type User struct {
    Username string
    Sid string
    Stats string "."
}

type StatList map[string]string

type Stat struct {
    Id int
    Value int
}

func init() {
    http.HandleFunc("/", stat)
    http.HandleFunc("/update", stat)
    http.HandleFunc("/stat", stat)
    http.HandleFunc("/setting", stat)
}

var c appengine.Context
var client *http.Client
var writer http.ResponseWriter

func stat(w http.ResponseWriter, r *http.Request){
    username, sid, jsonText := r.FormValue("username"), r.FormValue("sid"), r.FormValue("stats")
    var user User
    //var statkey *datastore.Key
    c = appengine.NewContext(r)
    client = urlfetch.Client(c)
    writer = w
    userKey, err := datastore.NewQuery("User").Filter("Username =", username).Limit(1).Run(c).Next(&user)
    if(err == datastore.Done){
        if(!validateUser(username, sid)){
            return
        }
        user = User{
            Username: username,
        }
        userKey, _ = datastore.Put(c, datastore.NewIncompleteKey(c, "User", nil), &user)
    }
    if(sid != user.Sid){
        if(!validateUser(username, sid)){
            return
        }
        user.Sid = sid
    }
    var stats StatList

    if(user.Stats == "."){
        stats = make(StatList)
        user.Stats = "{}"
    }else if(user.Stats != ""){
        _ = json.Unmarshal([]byte(user.Stats), &stats)
    }
    if(user.Stats == ""){
        stats = make(StatList)
        statIterate := datastore.NewQuery("Stat").Ancestor(userKey).Run(c)
        var curStat Stat
        for {
            curStatKey, err := statIterate.Next(&curStat)
            if(err == datastore.Done){
                break
            }
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            stats[fmt.Sprintf("%d", curStat.Id)] = fmt.Sprintf("%d", curStat.Value)
            datastore.Delete(c, curStatKey)
        }
    }
    var inputData map[string]interface{}
    err = json.Unmarshal([]byte(jsonText), &inputData)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        //fmt.Fprintf(w, "%s, %s, %s\n", username, sid, string(contents))
        return
    }
    for jkey, jval := range inputData {
        var ival string
        switch jval.(type){
            case int:
                ival = fmt.Sprintf("%d", jval.(int))
                break
            case string:
                ival = jval.(string)
                break
            case float64:
                ival = fmt.Sprintf("%d", int(jval.(float64)))
                break
        }
        stats[jkey] = ival
    }
    outstring, _ := json.Marshal(stats)
    fmt.Fprintf(w, "%s", outstring)
    user.Stats = string(outstring)
    datastore.Put(c, userKey, &user)
}

func validateUser(username string, sid string) bool {
    resp, err := client.Get("http://session.minecraft.net/game/joinserver.jsp?user=" + username + "&sessionId=" + sid + "&serverId=global_achievements");
    if err != nil {
        http.Error(writer, err.Error(), http.StatusBadGateway)
        //fmt.Fprintf(w, "%s, %s, %s\n", username, sid, string(contents))
        return false
    }
    defer resp.Body.Close()
    bac, _ := ioutil.ReadAll(resp.Body)
    contents := string(bac)
    if contents != "OK" {
        http.Error(writer, contents, http.StatusUnauthorized)
        return false
    }
    return true
}