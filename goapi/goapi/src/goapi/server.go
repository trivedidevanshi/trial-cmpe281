package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	// b64 "encoding/base64"
	"github.com/codegangsta/negroni"
	"github.com/rs/cors"
	"github.com/gorilla/mux"
	"github.com/unrolled/render"
	"github.com/satori/go.uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
) 
// MongoDB Config 
//public: 52.9.187.122 //devu: 10.0.0.152 //192.168.99.100
//kesha: 10.0.1.254
var mongodb_server = "192.168.99.100:27017"
var mongodb_database = "records" 
var mongodb_collection_user = "user"
var mongodb_collection_cart = "cart"
var mongodb_collection_items = "items"
var mongodb_collection_orders = "orders"
var mongodb_collection_userid = "userid"
 

// NewServer configures and returns a Server.
func NewServer() *negroni.Negroni {
	formatter := render.New(render.Options{
		IndentJSON: true,
	})
	corsObj := cors.New(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
        AllowedHeaders: []string{"Accept", "content-type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
	})
	
	n := negroni.Classic()
	mx := mux.NewRouter()
	initRoutes(mx, formatter)
	n.Use(corsObj)
	n.UseHandler(mx)
	return n
}

// API Routes
func initRoutes(mx *mux.Router, formatter *render.Render) {
	mx.HandleFunc("/upload",uploadFile(formatter)).Methods("POST") //items
	mx.HandleFunc("/download/{path}",downloadFile(formatter)).Methods("GET") 

	mx.HandleFunc("/ping", pingHandler(formatter)).Methods("GET")
	mx.HandleFunc("/signup", signupHandler(formatter)).Methods("POST")  //user done
	mx.HandleFunc("/login", loginHandler(formatter)).Methods("POST")  //user done
 
 	mx.HandleFunc("/itembyid/{itemid}", getItemByIDInventoryHandler(formatter)).Methods("GET") //items
	mx.HandleFunc("/addonecart/{itemid}", addonetoCartHandler(formatter)).Methods("PUT")  //cart
	mx.HandleFunc("/deductonecart/{itemid}", deductonetoCartHandler(formatter)).Methods("PUT")  //cart
	mx.HandleFunc("/insertuserid/{userid}", insertuserIDHandler(formatter)).Methods("PUT") //userid

	mx.HandleFunc("/getallcart/{userid}", cartAllDataHandler(formatter)).Methods("POST") //cart
	mx.HandleFunc("/inventory", inventoryAllDataHandler(formatter)).Methods("GET") //items
	mx.HandleFunc("/orders/{userid}", ordersAllDataHandler(formatter)).Methods("GET") //orders
	mx.HandleFunc("/admin", postDataForViewHandler(formatter)).Methods("POST") //items

	mx.HandleFunc("/cart", 	addnewitemtoCartHandler(formatter)).Methods("POST") //items
	mx.HandleFunc("/insertcart", cartinsertHandler(formatter)).Methods("POST") //cart

	mx.HandleFunc("/orders/{userid}", placeOrderHandler(formatter)).Methods("POST") //cart
	mx.HandleFunc("/postplaceorder", postplaceorderHandler(formatter)).Methods("POST") //orders

}

// Helper Functions
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
    (*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
    (*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

// API Ping Handler
func pingHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		formatter.JSON(w, http.StatusOK, struct{ Test string }{"API version 1.0 alive!"})
	}
}

const (
	S3_REGION = "us-west-1"
	S3_BUCKET = "fashiop-images"
)


func uploadFile(formatter *render.Render) http.HandlerFunc {
return func (w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, "Uploading File")
	r.ParseMultipartForm(32 << 20)

	formdata := r.MultipartForm

	files := formdata.File["photos"]
	fmt.Println("File", len(files))
	//log(len(files))

	var len int = len(files)
	// fmt.Println("File===>>>>>>", len)
	names := make([]string,len)
	s, err := session.NewSession(&aws.Config{Region: aws.String(S3_REGION)})
	if err != nil {
		log.Fatal(err)
	}
	for i, _ := range files {
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		defer file.Close()
		fmt.Println("File", files[i].Size)
 
		var size int64 = files[i].Size
		buffer := make([]byte, size)
		file.Read(buffer)
		uuid := uuid.NewV4()
		files[i].Filename = uuid.String()
		_, err = s3.New(s).PutObject(&s3.PutObjectInput{
			Bucket:               aws.String(S3_BUCKET),
			Key:                  aws.String(files[i].Filename),
			ACL:                  aws.String("private"),
			Body:                 bytes.NewReader(buffer),
			ContentLength:        aws.Int64(size),
			ContentType:          aws.String(http.DetectContentType(buffer)),
			ContentDisposition:   aws.String("attachment"),
			ServerSideEncryption: aws.String("AES256"),
		})

		names[i] = "http://d1juoluhxbb7ba.cloudfront.net/"+files[i].Filename
	}
	formatter.JSON(w, http.StatusOK, names)

}
}

func downloadFile(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		fmt.Printf("paraparamsms[id]=%s \n", params["path"])
		buff := &aws.WriteAtBuffer{}
		sess, _ := session.NewSession(&aws.Config{
			Region: aws.String(S3_REGION)},
		)

		downloader := s3manager.NewDownloader(sess)

		numBytes, err := downloader.Download(buff,
			&s3.GetObjectInput{
				Bucket: aws.String(S3_BUCKET),
				Key:    aws.String(params["path"]),
			})

		if err != nil {
			log.Fatalf("Unable to download item ")
		}
		// sEnc := b64.RawStdEncoding.EncodeToString(buff.Bytes())

		fmt.Println("Downloaded",numBytes, "bytes")
		w.Write(buff.Bytes())
		
	//	formatter.JSON(w, http.StatusOK, buff.Bytes())
	}
}



// API Signup - Create user
func signupHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
		if err != nil {
			panic(err)
		}
		defer session.Close()
		session.SetMode(mgo.Monotonic, true)

		var user User
		if err := json.NewDecoder(req.Body).Decode(&user); err != nil {
			fmt.Println(" Error: ", err)
			formatter.JSON(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		uuid := uuid.NewV4()
		user.User_id = uuid.String()
		fmt.Println("user.User_id: ", user.User_id)
		fmt.Println("req.Body: ", req.Body)
		fmt.Println("User signup Details:", user)

		c := session.DB(mongodb_database).C(mongodb_collection_user)
 
		var res Response 
		var result bson.M
		
		err = c.Find(bson.M{"email":user.Email}).One(&result)
		if err == nil {
			res.Status=-1
			res.Data="User with this email already exists."
			fmt.Println("User Details: ", err)
			formatter.JSON(w, http.StatusOK, res)
			return
		}

		if err := c.Insert(&user); err != nil {
			fmt.Println(" Error: ", err)
			formatter.JSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		formatter.JSON(w, http.StatusCreated, user)
	}
}

// POST LOGIN /login/{email}

func loginHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
		if err != nil {
			panic(err)
		}
		defer session.Close()
		session.SetMode(mgo.Monotonic, true)
		var user User
		if err := json.NewDecoder(req.Body).Decode(&user); err != nil {
			fmt.Println(" Error: ", err)
			formatter.JSON(w, http.StatusBadRequest, "Invalid request payload")
			return
		} 
	 
		var result bson.M 
		c := session.DB(mongodb_database).C(mongodb_collection_user)
		err = c.Find(bson.M{"email":user.Email}).One(&result)
		var res Response
		if err != nil {
			res.Status=-1
			res.Data="User not Found"
			fmt.Println("User Details: ", err)
			formatter.JSON(w, http.StatusOK, res)
			return
		} 
		if(result["password"] != user.Password){
			res.Status=-1
			res.Data="User not Found"
			fmt.Println("User Password Error: ", err)
			formatter.JSON(w, http.StatusOK, res)
			return
		}  
		fmt.Println("Successful Login")   
		formatter.JSON(w, http.StatusOK, result)
	}
}

//getItemByIDInventoryHandler
func getItemByIDInventoryHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["itemid"])
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
        c := session.DB(mongodb_database).C(mongodb_collection_items)
        var result bson.M
        err = c.Find(bson.M{"itemid" : params["itemid"]}).One(&result)
        if err != nil {
                log.Fatal(err)
        }
        fmt.Println("Inventory  Data By ID:", result )
		formatter.JSON(w, http.StatusOK, result)
	}
}

//
func addonetoCartHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["itemid"])
    	var m gumballMachine
    	_ = json.NewDecoder(req.Body).Decode(&m)		
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
        }
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
		c := session.DB(mongodb_database).C(mongodb_collection_cart)
		var result1 bson.M
        err = c.Find(bson.M{"itemid" : params["itemid"]}).One(&result1)
        if err != nil {
                log.Fatal(err)
		} 
		fmt.Println("Qunatity of Result: ", result1["quantity"])
		//fmt.Println("Updated Qunatity of Result: ", result1["quantity"]+1)
		query := bson.M{"itemid" : params["itemid"]}
		change := bson.M{"$set": bson.M{ "quantity" : result1["quantity"].(int)+1}}
        err = c.Update(query, change)
        if err != nil {
                log.Fatal(err)
        }
       	var result bson.M
        err = c.Find(bson.M{"itemid" : params["itemid"]}).One(&result)
        if err != nil {
                log.Fatal(err)
        }        
        fmt.Println("Cart Data:", result )
		formatter.JSON(w, http.StatusOK, result)
	}
}

//deductonetoCartHandler
func deductonetoCartHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["itemid"])
    	var m gumballMachine
    	_ = json.NewDecoder(req.Body).Decode(&m)		
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
        }
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
		c := session.DB(mongodb_database).C(mongodb_collection_cart)
		var result1 bson.M
        err = c.Find(bson.M{"itemid" : params["itemid"]}).One(&result1)
        if err != nil {
                log.Fatal(err)
		} 
		fmt.Println("Qunatity of Result: ", result1["quantity"])
		fmt.Println("Updated Qunatity of Result: ", result1["quantity"].(int)-1)
		query := bson.M{"itemid" : params["itemid"]}
		change := bson.M{"$set": bson.M{ "quantity" : result1["quantity"].(int)-1}}
        err = c.Update(query, change)
        if err != nil {
                log.Fatal(err)
        }
       	var result bson.M
        err = c.Find(bson.M{"itemid" : params["itemid"]}).One(&result)
        if err != nil {
                log.Fatal(err)
        }        
        fmt.Println("Cart Data:", result )
		formatter.JSON(w, http.StatusOK, result)
	}
}

// API to insert userid
func insertuserIDHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["userid"])
		defer session.Close()
		userdata := Trainer{params["userid"]}
		fmt.Println("User Data is",userdata)
        session.SetMode(mgo.Monotonic, true)
        c := session.DB(mongodb_database).C(mongodb_collection_userid)
        err = c.Insert(userdata)
        if err != nil {
                log.Fatal(err)
        }
		formatter.JSON(w, http.StatusOK, userdata)
	}
}

// API Get Order Status
// API Get Order Status
func cartAllDataHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}		
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["userid"])
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
        c := session.DB(mongodb_database).C("cart")
        var result []bson.M
        err = c.Find(bson.M{"userid" : params["userid"]}).All(&result)
        if err != nil {
                log.Fatal(err)
        }
        fmt.Println("Cart Data:", result )
		formatter.JSON(w, http.StatusOK, result)
	}
}

func inventoryAllDataHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session1, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
        }		
        defer session1.Close()
        session1.SetMode(mgo.Monotonic, true)
        c := session1.DB(mongodb_database).C(mongodb_collection_items)
        var result []Item
        err = c.Find(nil).All(&result)
        if err != nil {
                log.Fatal(err)
        }
		// fmt.Println("Inventory Data:", result )
		// fmt.Println("Result=>:", result[0] )
		// fmt.Println("Item path:", result[0].Itempath[0] )
		// fmt.Println("Item path res len: ", len(result))
		// fmt.Println("Item pat length:", len(result[0].Itempath) )

		

		// for i := 0; i < len(result); i++ {

		// 	for j := 0; j < len(result[i].Itempath); j++ {
		// 		var tempName string = result[i].Itempath[j];

		// fmt.Println("Item pat length:", tempName )
				

		// 		buff := &aws.WriteAtBuffer{}
		// 		sess, err3 := session.NewSession(&aws.Config{Region: aws.String(S3_REGION)},)
		// 		if err3!=nil{
		// 			log.Fatal(err3)
		// 		}
		// 		downloader := s3manager.NewDownloader(sess)

		// 		numBytes, err := downloader.Download(buff,
		// 			&s3.GetObjectInput{
		// 				Bucket: aws.String(S3_BUCKET),
		// 				Key:    aws.String(tempName),
		// 			})

		// 		if err != nil {
		// 			log.Fatalf("Unable to download item ")
		// 		}
				
		// 		fmt.Println("Downloaded", numBytes, "bytes")

		// 		result[i].temp[j]=buff.Bytes()
		// 		fmt.Println("Inventory Data:", result )
		// 		// w.Write(buff.Bytes())


		// 	}
		// }
		formatter.JSON(w, http.StatusOK, result)
	}
}
 
// API to view all Orders
func ordersAllDataHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}	
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["userid"])	
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
        c := session.DB(mongodb_database).C("orders")
        var result []bson.M
        err = c.Find(bson.M{"userid" : params["userid"]}).All(&result)
        if err != nil {
                log.Fatal(err)
        }
        fmt.Println("Order Data:", result)
		formatter.JSON(w, http.StatusOK, result)
	}
}

// API for admin
func postDataForViewHandler(formatter *render.Render)http.HandlerFunc{
	return func(w http.ResponseWriter,req *http.Request){
		
		uuid := uuid.NewV4()
		var itm Item
		 _ = json.NewDecoder(req.Body).Decode(&itm)
		itm.Itemid = uuid.String()
		fmt.Println(itm.Itempath)

		session, err := mgo.Dial(mongodb_server)
        if err != nil {
			fmt.Println("reached")
                panic(err)
        }
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
		c := session.DB(mongodb_database).C(mongodb_collection_items)
		
		err = c.Insert(itm)
		if err != nil {
			formatter.JSON(w, http.StatusNotFound, "Unable to add")
			return
		}
		fmt.Println("Added the item", itm)
		formatter.JSON(w, http.StatusOK, itm)
	}
}
 
// Add new item to cart
func addnewitemtoCartHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
        }
		defer session.Close()
		session.SetMode(mgo.Monotonic, true)
		//c := session.DB(mongodb_database).C("cart")
		d := session.DB(mongodb_database).C("items")

		var payment bson.M
		_ = json.NewDecoder(req.Body).Decode(&payment)
		
		fmt.Println("itemid",payment["itemid"])
		// params := mux.Vars(req)
		//  fmt.Printf("params[id]=%s \n", params["itemid"])
		var result1 bson.M
        err = d.Find(bson.M{"itemid" : payment["itemid"]}).One(&result1)
        if err != nil {
                log.Fatal("error is ",err)
		} 
		result1["quantity"]=payment["quantity"]
		result1["userid"]=payment["userid"]
		// var ar []byte=result1.([]byte)
		// r := bytes.NewReader(ar)
		jsonValue, _ := json.Marshal(result1)
		response, err := http.Post("http://localhost:3000/insertcart","application/json",bytes.NewBuffer(jsonValue))
		if err != nil {
			fmt.Printf("The HTTP request failed with error %s\n", err)
		} else {
			data, _ := ioutil.ReadAll(response.Body)
			fmt.Println(string(data))
		}

		// err = c.Insert(result1)
		// if err != nil {
		// 	formatter.JSON(w, http.StatusNotFound, "Create Payment Error")
		// 	return
		// }
		// fmt.Println("Create new payment:", payment)
		formatter.JSON(w, http.StatusOK, payment)
	}
}

// post method to insert data in cart
func cartinsertHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}
		var payment bson.M
		_ = json.NewDecoder(req.Body).Decode(&payment)
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
        c := session.DB(mongodb_database).C("cart")
        err = c.Insert(payment)
		if err != nil {
			formatter.JSON(w, http.StatusNotFound, "Create Pado buildyment Error")
			return
		}
		fmt.Println("Create new payment:", payment)
		formatter.JSON(w, http.StatusOK, payment)
	}
}

func placeOrderHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}	
		//uuid := uuid.NewV4()
		params := mux.Vars(req)
		 fmt.Printf("params[id]=%s \n", params["userid"])
        defer session.Close()
        session.SetMode(mgo.Monotonic, true)
		c := session.DB(mongodb_database).C("cart")
		//d := session.DB(mongodb_database).C("orders")
        var result []bson.M
		err = c.Find(bson.M{"userid" : params["userid"]}).All(&result) 
        if err != nil {
                log.Fatal(err)
        }
		fmt.Println("Cart Data:", result[0])
		fmt.Println("Cart Data:", len(result))
		jsonValue, _ := json.Marshal(result)
		response, err := http.Post("http://localhost:3000/postplaceorder","application/json",bytes.NewBuffer(jsonValue))
		if err != nil {
			fmt.Printf("The HTTP request failed with error %s\n", err)
		} else {
			data, _ := ioutil.ReadAll(response.Body)
			fmt.Println(string(data))
		}

		err = c.Remove(bson.M{"userid": params["userid"]})   //take the user id from url
        if err != nil {
                log.Fatal(err)
        }
		

		formatter.JSON(w, http.StatusOK, result)
	}
}
// API to post place order
func postplaceorderHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		session, err := mgo.Dial(mongodb_server)
        if err != nil {
                panic(err)
		}
		var result []bson.M
		_ = json.NewDecoder(req.Body).Decode(&result)

		// var result1 bson.M
		// json.Unmarshal(result, &result1)
        defer session.Close()
		session.SetMode(mgo.Monotonic, true)
		fmt.Println(result)
        c := session.DB(mongodb_database).C("orders")
        for i := 0; i < len(result); i++ {
			fmt.Println("Cart Data:", result[i])
				err = c.Insert(result[i])
				if err != nil {
					formatter.JSON(w, http.StatusNotFound, "Create Payment Error")
					return
				}
		}
		formatter.JSON(w, http.StatusOK, result)
	}
}
 /*
curl -d '{"email": "devu@gmail.com", "password": "admin", "type": "a", "fname": "devu", "lname": "trivedi", "phno": "1236568774", "address": "San Jose"}' -H "Content-Type: application/json" -X POST http://localhost:3000/signup
curl -d '{"email": "devu@gmail.com","password": "admin"}' -H "Content-Type: application/json" -X POST http://localhost:3000/login

	{"itemid": "1",       
			"price": 25,          
			"itemname": "CD",      
			"itemdesc": "def"  ,
			"itempath":"def.html",
			"quantity": 5,
			"sold" : 2}
 
curl -d '{"itemid": "3","quantity": 20}' -H "Content-Type: application/json" -X POST http://localhost:3000/cart


curl -d '{"itemid": "5cccac87b039e784b2124954","quantity": 1, "userid":1}' -H "Content-Type: application/json" -X POST http://localhost:3000/cart
/orders/{userid}

*/
