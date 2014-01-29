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
// chat$BIt20$KAjEv$9$k(BAncestorKey$B$rJV$9(B
func tinychatKey(c appengine.Context, key string) *datastore.Key {
	// func NewKey(c appengine.Context, kind, stringID string, intID int64, parent *Key) *Key
	if key == "" {
		return datastore.NewKey(c, "TinyChat", "default_tinychat", 0, nil)
	} else {
		return datastore.NewKey(c, "TinyChat", key, 0, nil)
	}
}

// chat$BIt20$K$$$k%a%s%P%j%9%H$KAjEv$9$k(BAncestorKey$B$rJV$9(B
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
	// chatkey$B$N%/%(%j$,6u$G$3$3$KMh$?>l9g!"(Bchatkey$B$r(B
	// $B$3$N%f!<%6$N(BID$B$H$9$k(B(=$B?7$7$$(Bchat$BIt20$r:n$k(B)
	if tkey == "" {
		tkey = u.ID
	}

	// channel $B$r(B uID+tkey$B$G:n$k(B
	// $B$I$N%A%c%C%HIt20$N$I$N%f!<%6$+$,7h$^$k(B
	tok, err := channel.Create(c, u.ID+tkey)
	if err != nil {
		http.Error(w, "Couldn't create Channel", http.StatusInternalServerError)
		c.Errorf("channel.Create: %v", err)
		return
	}

	// $B8=:_$N%A%c%C%HIt20$N:G?7$N(B
	// $BH/8@FbMF$r(B20$B7o<hF@(B
	q := datastore.NewQuery("message").Ancestor(tinychatKey(c, tkey)).Order("-Date").Limit(20)
	messages := make([]Message, 0, 20)
	if _, err := q.GetAll(c, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// u.ID$B$GIt20$N;22C<T%j%9%H$K<+J,$,$$$k$+$r8!:w(B
	q = datastore.NewQuery("Member").Ancestor(memberKey(c, tkey)).
		Filter("ID =", u.ID)
	members := make([]Member, 0, 1)
	if _, err := q.GetAll(c, &members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// $B8+$D$+$i$J$1$l$P<+J,$r%a%s%P%j%9%H$KDI2C(B
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

	// HTML$B=PNO$N$?$a$N=`Hw(B
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

// $BH/8@;~$N=hM}(B
// Javascript$B$N%/%i%$%"%s%H$+$i(B
// /submit?chatkey=(chatkey)[&msg=(msg)] $B$H$$$&%/%(%j$G%j%/%(%9%H$,$/$k(B
// msg$B$NFbMF$r%G!<%?%9%H%"$KJ]B8$7!"H/8@FbMF%j%9%H$r(Bchannel$B$r7PM3$7$F(B
// $BIt20$K$$$k$9$Y$F$N(BJavascript$B%/%i%$%"%s%H$K(BSendJSON$B$9$k(B
func submit(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	u := user.Current(c)
	tkey := r.FormValue("chatkey")

	// $BH/8@FbMF$r%G!<%?%9%H%"$KJ]B8$9$k(B
	stm := Message{
		Date:    time.Now(),
		Name:    u.String(),
		Content: r.FormValue("msg"),
	}

	log.Printf("tkey: %v, msg: %v\n", tkey, stm.Content)

	// $B%G!<%?%9%H%"$XH/8@FbMF$r(BPut
	stmkey := datastore.NewIncompleteKey(c, "message", tinychatKey(c, tkey))
	_, err := datastore.Put(c, stmkey, &stm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// $B%G!<%?%9%H%"$+$i8=:_$NIt20$K$$$k%f!<%6A40w$r<hF@$9$k(B
	q := datastore.NewQuery("Member").Ancestor(memberKey(c, tkey))
	members := make([]Member, 0, 20)
	if _, err := q.GetAll(c, &members); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("member: %v\n", members)

	// $BH/8@FbMF$r(B20$B7o<hF@(B
	q = datastore.NewQuery("message").Ancestor(tinychatKey(c, tkey)).Order("-Date").Limit(20)
	messages := make([]Message, 0, 20)
	if _, err := q.GetAll(c, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// $B$9$Y$F$N%f!<%6$KBP$7$F(BSendJSON$B$9$k(B
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
