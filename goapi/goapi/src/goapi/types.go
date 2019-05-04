package main

type gumballMachine struct {
	Id             	int 	
	CountGumballs   int    	
	ModelNumber 	string	    
	SerialNumber 	string	
}

type order struct {
	Id             	string 	
	OrderStatus 	string	
}

type User struct {
	User_id string `json:"user_id"`
	Email string `json:"email"`
	Password string `json:"password"`
	Type string `json:"type"`
	Fname string `json:"fname"`
	Lname string `json:"lname"`
	Phno string `json:"phno"`
	Address string `json:"address"`
 
}
type Response struct{
	Status int
	Data string
}

type Trainer struct {
	ID string 
}

type Item struct {
	Itemid string      
	Price string       
	Itemname string      
	Itemdesc string
	Itempath []string
	Quantity string
	Sold string
	temp [][]byte
}
 

var orders map[string] order

var Items map[string] Item
