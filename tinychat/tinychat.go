package tinychat

import (
	"html/template"
	"net/http"

	"appengine"
	"appengine/channel"
	"appengine/datastore"
	"appengine/user"
	"time"

	//"log"
)

type Message struct {
	Date    time.Time
	Name    string
	Content string
}

type Member struct {
	ID string
}

type Display struct {
	Token    string
	Me       string
	Chat_key string
	Messages []Message
}

var mainTemplate = template.Must(template.ParseFiles("main.html"))

///////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////
// chat部屋に相当するAncestorKeyを返す
func tinychatKey(c appengine.Context, key string) *datastore.Key {
	// func NewKey(c appengine.Context, kind, stringID string, intID int64, parent *Key) *Key
	if key == "" {
		return datastore.NewKey(c, "TinyChat", "default_tinychat", 0, nil)
	} else {
		return datastore.NewKey(c, "TinyChat", key, 0, nil)
	}
}

// chat部屋にいるメンバリストに相当するAncestorKeyを返す
func memberKey(c appengine.Context, key string) *datastore.Key {
	if key == "" {
		return datastore.NewKey(c, "Member", "default_member", 0, nil)
	} else {
		return datastore.NewKey(c, "Member", key, 0, nil)
	}
}

///////////////////////////////////////////////////////////
func init() {
	http.HandleFunc("/", main)
	http.HandleFunc("/submit", submit)
}

func main(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	u := user.Current(c) // assumes 'login: required' set in app.yaml

	tkey := r.FormValue("chatkey")
	// chatkeyのクエリが空でここに来た場合、chatkeyを
	// このユーザのIDとする(=新しいchat部屋を作る)
	if tkey == "" {
		tkey = u.ID
	}

	// channel を uID+tkeyで作る
	// どのチャット部屋のどのユーザかが決まる
	tok, err := channel.Create(c, u.ID+tkey)
	if err != nil {
		http.Error(w, "Couldn't create Channel", http.StatusInternalServerError)
		c.Errorf("channel.Create: %v", err)
		return
	}

	// 現在のチャット部屋の最新の
	// 発言内容を20件取得
	q := datastore.NewQuery("message").Ancestor(tinychatKey(c, tkey)).Order("-Date").Limit(20)
	messages := make([]Message, 0, 20)
	if _, err := q.GetAll(c, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// u.IDで部屋の参加者リストに自分がいるかを検索
	q = datastore.NewQuery("Member").Ancestor(memberKey(c, tkey)).
		Filter("ID =", u.ID)
	members := make([]Member, 0, 1)
	if _, err := q.GetAll(c, &members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 見つからなければ自分をメンバリストに追加
	if len(members) == 0 {
		m := Member{
			ID: u.ID,
		}
		key := datastore.NewIncompleteKey(c, "Member", memberKey(c, tkey))
		_, err := datastore.Put(c, key, &m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// HTML出力のための準備
	d := Display{
		Token:    tok,
		Me:       u.ID,
		Chat_key: tkey,
		Messages: messages,
	}

	err = mainTemplate.Execute(w, d)
	if err != nil {
		c.Errorf("mainTemplate: %v", err)
	}
}

// 発言時の処理
// Javascriptのクライアントから
// /submit?chatkey=(chatkey)[&msg=(msg)] というクエリでリクエストがくる
// msgの内容をデータストアに保存し、発言内容リストをchannelを経由して
// 部屋にいるすべてのJavascriptクライアントにSendJSONする
func submit(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	u := user.Current(c)
	tkey := r.FormValue("chatkey")

	// 発言内容をデータストアに保存する
	stm := Message{
		Date:    time.Now(),
		Name:    u.String(),
		Content: r.FormValue("msg"),
	}

	log.Printf("tkey: %v, msg: %v\n", tkey, stm.Content)

	// データストアへ発言内容をPut
	stmkey := datastore.NewIncompleteKey(c, "message", tinychatKey(c, tkey))
	_, err := datastore.Put(c, stmkey, &stm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// データストアから現在の部屋にいるユーザ全員を取得する
	q := datastore.NewQuery("Member").Ancestor(memberKey(c, tkey))
	members := make([]Member, 0, 20)
	if _, err := q.GetAll(c, &members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("member: %v\n", members)

	// 発言内容を20件取得
	q = datastore.NewQuery("message").Ancestor(tinychatKey(c, tkey)).Order("-Date").Limit(20)
	messages := make([]Message, 0, 20)
	if _, err := q.GetAll(c, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// すべてのユーザに対してSendJSONする
	d := Display{
		Token:    "",
		Me:       u.ID,
		Chat_key: tkey,
		Messages: messages,
	}
	for _, member := range members {
		err := channel.SendJSON(c, member.ID+tkey, d)
		if err != nil {
			c.Errorf("sending data: %v", err)
		}
	}

	//log.Printf("hello")
}
