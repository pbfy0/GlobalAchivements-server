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
    http.HandleFunc("/", update)
    http.HandleFunc("/update", update)
}

func update(w http.ResponseWriter, r *http.Request){
    username, sid, jsonText := r.FormValue("username"), r.FormValue("sid"), r.FormValue("stats")
    var user User
    //var statkey *datastore.Key
    c := appengine.NewContext(r)
    userKey, err := datastore.NewQuery("User").Filter("Username =", username).Limit(1).Run(c).Next(&user)
    if(err == datastore.Done){
        user = User{
            Username: username,
        }
        userKey, _ = datastore.Put(c, datastore.NewIncompleteKey(c, "User", nil), &user)
    }
    if(sid != user.Sid){
        client := urlfetch.Client(c)
        resp, err := client.Get("http://session.minecraft.net/game/joinserver.jsp?user=" + username + "&sessionId=" + sid + "&serverId=global_achievements");
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadGateway)
            //fmt.Fprintf(w, "%s, %s, %s\n", username, sid, string(contents))
            return
        }
        defer resp.Body.Close()
        bac, _ := ioutil.ReadAll(resp.Body)
        contents := string(bac)
        if contents != "OK" {
            http.Error(w, contents, http.StatusUnauthorized)
            return
        }
        user.Sid = sid
    }
    var stats StatList
    if(user.Stats == ""){
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
    if(user.Stats == "."){
        stats = make(StatList)
        user.Stats = "{}"
    }else{
        _ = json.Unmarshal([]byte(user.Stats), &stats)
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
                ival, _ = jval.(string)
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