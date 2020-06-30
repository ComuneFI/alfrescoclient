package alfrescoclient

// author Cristian Lorenzetto
import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"strconv"
)

//http://alfresco-test.comune.intranet:8080/alfresco/api/-default-/public/cmis/versions/1.1/atom
const LOGIN_URL = "alfresco/service/api/login"

const API_PATH = "alfresco/api/-default-/public/alfresco/versions/1/nodes/"
const PROTO = "http://"

type AlfrescoClient struct {
	ticket  string
	client  *resty.Client
	baseUrl string
}

type ticketData struct {
	Ticket string `json:"ticket"`
}
type authResponse struct {
	Data *ticketData `json:"data"`
}
type errorReturnData struct {
	Error ErrorData `json:"error"`
}
type ErrorData struct {
	ErrorKey       string `json:"errorKey"`
	StatusCode     uint   `json:"statusCode"`
	BriefSummary   string `json:"briefSummary"`
	StackTrace     string `json:"stackTrace"`
	DescriptionURL string `json:"descriptionURL"`
	LogId          string `json:"logId"`
}
type Condition struct {
	op1     *selectwhere
	op2     *Condition
	operand uint8
}
type selectwhere struct {
	name  string
	op    string
	value interface{}
}

func (c *selectwhere) toString() string {
	str := fmt.Sprintf("%v", c.value)
	return c.name + c.op + str
}
func NewCondition(k, op string, v interface{}) *Condition {
	c := Condition{}
	c.Filter(k, op, v)
	return &c
}

func (c *Condition) Filter(k, op string, v interface{}) *Condition {

	c.op1 = &selectwhere{name: k, op: op, value: v}
	c.operand = 0
	return c
}

func (c *Condition) And(c2 *Condition) *Condition {
	c.op2 = c2
	c.operand = 1
	return c
}
func (c *Condition) Or(c2 *Condition) *Condition {
	c.op2 = c2
	c.operand = 2
	return c
}
func (c *Condition) toString() string {
	str := c.op1.toString()
	if c.operand == 1 {
		str = str + " and ( " + c.op2.toString() + " )"
	} else if c.operand == 2 {
		str = str + " or ( " + c.op2.toString() + " )"
	}

	return "( " + str + " )"
}

func (e *ErrorData) Error() string {
	return e.ErrorKey
}

type PaginationData struct {
	Count        uint `json:"count"`
	HasMoreItems bool `json:"hasMoreItems"`
	TotalItems   uint `json:"totalItems"`
	SkipCount    uint `json:"skipCount"`
	MaxItems     uint `json:"maxItems"`
}

type ListResponse struct {
	List ListData `json:"list"`
}

type UserInfoData struct {
	DisplayName string `json:"displayName"`
	Id          string `json:"id"`
}

type ContentTypeData struct {
	MimeType     string `json:"mimeType"`
	MimeTypeName string `json:"mimeTypeName"`
	Encoding     string `json:"encoding"`
	SizeInBytes  uint32 `json:"sizeInBytes"`
}

type returnNodeData struct {
	Entry NodeData `json:"entry"`
}

type inData struct {
	Name       string      `json:"name"`
	Properties interface{} `json:"properties"`
	NodeType   string      `json:"nodeType"`
}

type EntryData struct {
	Entry NodeData `json:"entry"`
}

type NodeData struct {
	Id             string          `json:"id"`
	IsFolder       bool            `json:"isFolder"`
	Name           string          `json:"name"`
	ParentId       string          `json:"parentId"`
	NodeType       string          `json:"nodeType"`
	ModifiedAt     string          `json:"modifiedAt"`
	CreatedAt      string          `json:"createdAt"`
	ModifiedByUser UserInfoData    `json:"modifiedByUser"`
	CreatedByUser  UserInfoData    `json:"createdByUser"`
	Content        ContentTypeData `json:"content"`
}

type listResult struct {
	List ListData `json:"list"`
}

type ListData struct {
	Pagination PaginationData `json:"pagination"`
	Entries    []EntryData    `json:"entries"`
}

type SortType string

const (
	DESC SortType = "DESC"
	ASC  SortType = "ASC"
)

func (s SortType) String() string {
	return string(s)
}

func (dd *AlfrescoClient) Init(host string, port uint, username, password string) error {
	url := host + ":" + strconv.FormatUint(uint64(port), 10) + "/"
	dd.baseUrl = url
	dd.client = resty.New()
	login := PROTO + url + LOGIN_URL
	response := authResponse{}
	rr, err := dd.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(`{"username":"` + username + `", "password":"` + password + `"}`).
		SetResult(&response).
		Post(login)
	if err == nil {
		if response.Data != nil {
			dd.ticket = base64.StdEncoding.EncodeToString([]byte(response.Data.Ticket))

		} else {
			err = errors.New("invalid response")
		}
	}
	if rr == nil {
		return nil
	}
	return err

}

func (dd *AlfrescoClient) GetNodeContent(id string) (io.ReadCloser, error) {
	url := PROTO + dd.baseUrl + API_PATH + id + "/content"
	e := errorReturnData{}
	resp, err := dd.client.R().SetError(&e).SetDoNotParseResponse(true).SetHeader("Authorization", "Basic "+dd.ticket).Get(url)
	if err != nil {
		return nil, errors.New("invalid response")
	}
	if resp.StatusCode() != 200 {
		return nil, &e.Error
	}
	//fmt.Println("--->" +string(resp.Body()))
	return resp.RawBody(), err

}

func (dd *AlfrescoClient) GetNodeMetadata(id string) (*NodeData, error) {
	url := PROTO + dd.baseUrl + API_PATH + id
	e := errorReturnData{}
	success := returnNodeData{}

	resp, err := dd.client.R().SetResult(&success).SetError(&e).SetHeader("Authorization", "Basic "+dd.ticket).Get(url)
	if err != nil {
		return nil, errors.New("invalid response")
	}
	if resp.StatusCode() != 200 {
		return nil, &e.Error
	}
	//fmt.Println("--->" +string(resp.Body()))
	return &success.Entry, err

}

func (dd *AlfrescoClient) DeleteNode(id string) error {
	e := errorReturnData{}
	resp, err := dd.client.R().SetHeader("Authorization", "Basic "+dd.ticket).SetError(&e).Delete(PROTO + dd.baseUrl + API_PATH + id)
	if resp.StatusCode() != 204 {
		return &e.Error
	}
	return err

}

func (dd *AlfrescoClient) ListNodes(parent string, skip, max uint, cond *Condition, sort *map[string]SortType) (*ListData, error) {
	list := listResult{}
	e := errorReturnData{}
	m := map[string]string{
		"skipCount": string(skip),
		"maxitems":  string(max),
	}
	if cond != nil {
		m["where"] = cond.toString()
	}
	if sort != nil {
		var str string = ""
		i := 0
		for k, v := range *sort {
			if i > 0 {
				str += ","
			}
			str += k + "=" + v.String()
			i++
		}
		m["orderBy"] = str
	}
	resp, err := dd.client.R().SetHeader("Authorization", "Basic "+dd.ticket).SetResult(&list).SetError(&e).SetPathParams(m).Get(PROTO + dd.baseUrl + API_PATH + parent + "/children")
	if resp.StatusCode() != 200 {
		return nil, &e.Error
	}
	//fmt.Println("--->" +string(resp.Body()))
	return &list.List, err

}

func (dd *AlfrescoClient) CreateNode(parentId, name string, properties interface{}) (*NodeData, error) {
	e := errorReturnData{}
	success := returnNodeData{}
	p := inData{
		Name:       name,
		Properties: properties,
		NodeType:   "cm:content",
	}
	url := PROTO + dd.baseUrl + API_PATH + parentId + "/children"

	resp, err := dd.client.R().SetHeader("Authorization", "Basic "+dd.ticket).
		SetHeader("Content-Type", "application/json").
		SetBody(p).
		SetResult(&success).SetError(&e).
		Post(url)
	if resp.StatusCode() != 201 {
		return nil, &e.Error
	} else if err == nil {
		//fmt.Println(string(resp.Body()))
		return &success.Entry, err
	}
	return nil, err

}

func (dd *AlfrescoClient) SaveContent(id, name string, reader io.Reader) error {
	e := errorReturnData{}
	resp, err := dd.client.R().SetHeader("Authorization", "Basic "+dd.ticket).
		SetHeader("Content-Type", "application/octet-stream").
		SetBody(reader).
		SetError(&e).
		Put(PROTO + dd.baseUrl + API_PATH + id + "/content")
	if resp.StatusCode() != 200 {
		// fmt.Println(string(resp.Body()))
		return &e.Error
	}

	return err

}

func (dd *AlfrescoClient) UpdateMetadata(id, name string, properties interface{}) error {
	e := errorReturnData{}

	p := inData{
		Name:       name,
		Properties: properties,
	}
	resp, err := dd.client.R().SetBody(p).SetError(&e).SetHeader("Authorization", "Basic "+dd.ticket).
		Put(PROTO + dd.baseUrl + API_PATH + id)

	if resp.StatusCode() != 200 {
		//fmt.Println(string(resp.Body()))
		return &e.Error
	}
	return err

}
