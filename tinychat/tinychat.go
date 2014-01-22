package tinychat

import (
	"html/template"
	"net/http"

	"appengine"
	"appengine/datastore"
	"appengine/channel"
	"appengine/user"
	"time"

	"log"
)

type Statement struct {
	Date       time.Time
	Name       string
	Content    string
}

type Member struct {
	ID        string
}

type Display struct {
	Token      string
	Me         string
	Chat_key   string
	Messages   []Statement
}


var mainTemplate = template.Must(template.ParseFiles("main.html"))

///////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////
func init() {
	http.HandleFunc("/", main)
	http.HandleFunc("/submit", submit)
}

func tinychatKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "TinyChat", "default_tinychat", 0, nil)
}

func memberKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Member", "default_member", 0, nil)
}

func main(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	u := user.Current(c) // assumes 'login: required' set in app.yaml

	tkey := r.FormValue("chatkey")
	// もし、だれもログインしていなければ、chatkeyを
	// 最初にログインしたユーザのIDとする
	if tkey == "" {
		tkey = u.ID
	}

	log.Println("hoge")

	// channel を uID+tkeyで作る
	tok, err := channel.Create(c, u.ID+tkey)
	if err != nil {
		http.Error(w, "Couldn't create Channel", http.StatusInternalServerError)
		c.Errorf("channel.Create: %v", err)
		return
	}

	// 発言内容を20件取得
	q := datastore.NewQuery("message").Ancestor(tinychatKey(c)).Order("-Date").Limit(20)
	messages := make([]Statement, 0, 20)
	if _, err := q.GetAll(c, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("piyo")

	// u.IDで参加者リストを検索
	q = datastore.NewQuery("Member").Ancestor(memberKey(c)).
			Filter("ID =", u.ID)
	members := make([]Member, 0, 1)

	log.Println("fuga")

	if _, err := q.GetAll(c, &members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m := Member{
		ID: u.ID,
	}
	log.Printf("morua: %v, %v", members, m)

	// 見つからなければ追加
	if len(members) == 0 {
		key := datastore.NewIncompleteKey(c, "Member", memberKey(c))
		_, err := datastore.Put(c, key, &m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	log.Println("moga")

	d := Display {
		Token:		tok,
		Me:			u.ID,
		Chat_key:	tkey,
		Messages:	messages,
	}

	err = mainTemplate.Execute(w, d)
	if err != nil {
		c.Errorf("mainTemplate: %v", err)
	}
}

// 発言時の処理
// /submit?chatkey=(chatkey)[&msg=(msg)] というクエリでくる
func submit(w http.ResponseWriter, r *http.Request) {

	log.Println("submit is called");

	c := appengine.NewContext(r)
	u := user.Current(c)
	key := r.FormValue("chatkey")

	// 発言内容をデータストアに保存する
	stm := Statement {
		Date:		time.Now(),
		Name:		u.String(),
		Content:	r.FormValue("msg"),
	}

	log.Printf("key: %v, msg: %v\n", key, stm.Content);

	stmkey := datastore.NewIncompleteKey(c, "message", tinychatKey(c))
	_, err := datastore.Put(c, stmkey, &stm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// データストアから現在のユーザ全員を取得する
	q := datastore.NewQuery("Member").Ancestor(memberKey(c))
	members := make([]Member, 0, 20)
	if _, err := q.GetAll(c, &members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("member: %v\n", members);

	// 発言内容を20件取得
	q = datastore.NewQuery("message").Ancestor(tinychatKey(c)).Order("-Date").Limit(20)
	messages := make([]Statement, 0, 20)
	if _, err := q.GetAll(c, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d := Display {
		Token:		"",
		Me:			u.ID,
		Chat_key:	key,
		Messages:	messages,
	}

	// すべてのユーザに対してSendJSONする
	for _, member := range members {
		err := channel.SendJSON(c, member.ID+key, d)
		if err != nil {
			c.Errorf("sending data: %v", err)
		}
	}

	log.Println("submit function end")

}

