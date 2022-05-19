package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	amf_crypto "github.com/mft-labs/amf_crypto"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//fLwkiOkxASG3VrDD1dr9
//fLwkiOkxASG3VrDD1dr9

type apiMgr struct {
	username   string
	password   string
	apiurl     string
	codelist   string
	infile     string
	bkpfile    string
	bkpdir     string
	bkpfileptr *excelize.File
	config     *ini.File
	errorsList []string
}

type codelistItem struct {
	active       string
	senderCode   string
	receiverCode string
	description  string
	text         [10]string
}

/*
[
  {
    "_id": "Zydus_SAP_Cust|||1",
    "codeListName": "Zydus_SAP_Cust",
    "versionNumber": 1,
    "createDate": "2019-05-16T17:08:52.000+0000",
    "userName": "apiuser",
    "listStatus": 1,
    "codes": [
      {
        "senderCode": "test1",
        "receiverCode": "testre",
        "description": "no test"
      }
    ]
  }
]
*/

type Code struct {
	senderCode   string
	receiverCode string
	description  string
	text1        string `json:"item,omitempty"`
	text2        string `json:"item,omitempty"`
	text3        string `json:"item,omitempty"`
	text4        string `json:"item,omitempty"`
	text5        string `json:"item,omitempty"`
	text6        string `json:"item,omitempty"`
	text7        string `json:"item,omitempty"`
	text8        string `json:"item,omitempty"`
	text9        string `json:"item,omitempty"`
}

type CodeListItem struct {
	_id           string    `json:"item,omitempty"`
	codeListName  string    `json:"item,omitempty"`
	versionNumber float64   `json:"item,omitempty"`
	createDate    time.Time `json:"item,omitempty"`
	userName      string    `json:"item,omitempty"`
	listStatus    float64   `json:"item,omitempty"`
	codes         []Code    `json:"list"`
}

type CodeListItemResponse struct {
	Collection []CodeListItem
}

func formattedCurTimeStamp(format string) string {
	t := time.Now()
	return t.Format(format)
}

func (mgr *apiMgr) addError(errmsg string) {
	mgr.errorsList = append(mgr.errorsList, errmsg)
}
func (mgr *apiMgr) init() error {
	sec, err := mgr.config.GetSection("DEFAULT")
	timestamp_format := "20060102_150405"
	if err != nil {
		mgr.addError("ERROR: Missing DEFAULT section")
	}
	mgr.username = sec.Key("username").String()
	if mgr.username == "" {
		mgr.addError("username")
	}
	mgr.password = decrypt(sec.Key("password").String())
	if mgr.password == "" {
		mgr.addError("password")
	}
	mgr.apiurl = sec.Key("apiurl").String()
	if mgr.apiurl == "" {
		mgr.addError("apiurl")
	}

	mgr.bkpdir = sec.Key("backupdir").String()
	if mgr.bkpdir == "" {
		mgr.bkpdir = "codelist-backup"
	}

	if len(mgr.errorsList) > 0 {
		return fmt.Errorf("Missing keys")
	}

	err = mgr.validateApiUrl()
	if err != nil {
		mgr.addError("ERROR: invalid apiurl or unable to reach the end-point")
		mgr.showErrors("")
		os.Exit(20001)
	}
	mgr.codelist = ""
	mgr.bkpfile = "bkp_codelist_" + formattedCurTimeStamp(timestamp_format) + ".xlsx"
	mgr.bkpfileptr = excelize.NewFile()
	if _, err := os.Stat(mgr.bkpdir); os.IsNotExist(err) {
		fmt.Printf("Creating backup directory: %s\n", mgr.bkpdir)
		os.MkdirAll(mgr.bkpdir, os.ModePerm)
	}

	return nil

}

func (mgr *apiMgr) showErrors(title string) {
	fmt.Println("AMF CodeList Manager")
	fmt.Println("======================================================================")
	fmt.Println("Please fix the following errors and run again.")
	if title != "" {
		fmt.Printf("%s\n", title)
	}
	for _, errin := range mgr.errorsList {
		fmt.Println(errin)
	}
}

func (mgr *apiMgr) validateApiUrl() error {
	// /B2BAPIs/svc/codelists/?locale=en_US&_range=0-999&_accept=application%2Fjson&_contentType=application%2Fjson&_method=HEAD
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	queryParams := "?locale=en_US&_range=0-999&_accept=application%2Fjson&_contentType=application%2Fjson&_method=HEAD"
	req, err := http.NewRequest("HEAD", mgr.apiurl+"/B2BAPIs/svc/codelists/"+queryParams, nil)
	req.SetBasicAuth(mgr.username, mgr.password)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		//fmt.Printf("%s (1)",err)
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("%s (2)",err);
		return fmt.Errorf("ERROR: invalid api url")
	}
	if 200 != resp.StatusCode {
		//fmt.Printf("ERROR: Response Code -> %d",resp.StatusCode)
		return fmt.Errorf("ERROR: invalid api url")
	}
	return nil
}

func (mgr *apiMgr) runUpdate() error {
	//fmt.Println("Running bulk update using "+ mgr.infile +" for code list "+mgr.codelist+" using account "+mgr.username)
	fmt.Println("Sterling B2B Integrator \"Code Lists\" are being updated using \"" + mgr.username + "\" account and \"" + mgr.infile + "\"")
	f, err := excelize.OpenFile(mgr.infile)
	if err != nil {
		return fmt.Errorf("ERROR - Invalid input file [%s]", mgr.infile)
	}

	backupDone := false
	for _, name := range f.GetSheetMap() {
		mgr.codelist = name
		if mgr.codelist != "Instructions" {
			mgr.backupCodelist()
			backupDone = true
		}
	}
	if !backupDone {
		mgr.addError("ERROR: invalid input document or CodeList(s) not found")
		mgr.showErrors("")
		os.Exit(20002)
	}
	mgr.bkpfileptr.DeleteSheet("Sheet1")
	ok := mgr.bkpfileptr.SaveAs(mgr.bkpdir + "/" + mgr.bkpfile)

	codelistFailedArr := make([]string, 0)
	if ok == nil {
		fmt.Println("A backup file \"" + mgr.bkpfile + "\" has been created.")
		//fmt.Println("Backup file created for codelist successfully, continuing for bulk update of codelist")
		//fmt.Println("Going to clean the codelists")

		//Removed delete of codelists as per discussion with Raja on 29th May, 2019
		f2, err := excelize.OpenFile(mgr.bkpdir + "/" + mgr.bkpfile)
		if err != nil {
			fmt.Println("Error occurred while trying to clean up Code Lists", err)
			os.Exit(3)
		}
		//warning:=false;
		for _, name := range f2.GetSheetMap() {
			if name != "Sheet1" {
				err := mgr.deleteCodelist(name)
				if err != nil {
					fmt.Println("Unable to delete the Code List: \"" + name + "\"")
					fmt.Println("It is recommended to remove all versions of this Code List: \"" + name + "\" manually and run the script again.")
					fmt.Println("Continuing with remaining Code Lists")
					//os.Exit(2)
					//warning=true;
					codelistFailedArr = append(codelistFailedArr, name)
				}
			}
		}

	} else {
		return fmt.Errorf("Failed to create backup file [%s]", mgr.bkpfile)
	}
	for _, name := range f.GetSheetMap() {
		codelistErrors := make([]string, 0)
		mgr.codelist = name
		if mgr.codelistFailed(codelistFailedArr) {
			continue
		}
		//mgr.backupCodelist()
		if mgr.codelist != "Instructions" {
			//fmt.Println("Updating codelist ->  "+mgr.codelist)
			rows := f.GetRows(mgr.codelist)
			var codelist = make([]map[string]string, 0)
			rownum := 0
			for _, row := range rows {
				rownum = rownum + 1
				clitem := &codelistItem{}
				clitem.active = row[0]
				clitem.senderCode = row[1]
				clitem.receiverCode = row[2]
				clitem.description = row[3]
				for i := 1; i <= 9; i++ {
					if len(row) >= i+3 {
						clitem.text[i-1] = row[i+3]
					} else {
						clitem.text[i-1] = ""
					}

				}
				//clist = append(clist,clitem)
				if (clitem.active == "Yes") && (clitem.senderCode == "" || clitem.receiverCode == "") {
					codelistErrors = append(codelistErrors, "ERROR: invalid data (sendercode or receivercode missing) ignoring at row "+strconv.Itoa(rownum))
					continue
				}
				msg := map[string]string{
					"senderCode":   clitem.senderCode,
					"receiverCode": clitem.receiverCode,
					"description":  clitem.description,
					"text1":        clitem.text[0],
					"text2":        clitem.text[1],
					"text3":        clitem.text[2],
					"text4":        clitem.text[3],
					"text5":        clitem.text[4],
					"text6":        clitem.text[5],
					"text7":        clitem.text[6],
					"text8":        clitem.text[7],
					"text9":        clitem.text[8],
				}
				if clitem.active == "Yes" {
					codelist = append(codelist, msg)
					/*val, err := json.Marshal(msg)
					        if err == nil {
						        //fmt.Println(string(val))
					        } else {
					        	fmt.Println(err)
					        }*/
				}
			}
			val2, err := json.Marshal(codelist)
			if err == nil {
				requestInfo := "{" + "\"codes\":" + string(val2) + ",\"listStatus\":1" + "}"
				//fmt.Println(requestInfo)
				_, err := mgr.BulkUpdate(requestInfo)
				if err != nil {
					//fmt.Println("Error occurred",err)
					if strings.Contains(err.Error(), "Codelist not found") {
						requestInfo = "{ \"codeListName\": \"" + mgr.codelist + "\", \"codes\":" + string(val2) + "}"
						//fmt.Println(requestInfo)
						_, err := mgr.CreateCodelist(requestInfo)
						if err != nil {
							fmt.Println("Error occurred", err)
						} else {
							//fmt.Println("Successfully created code list ",mgr.codelist)
							//fmt.Println(response2)
							fmt.Println(mgr.codelist, " created.")
						}
					}
				} else {
					//fmt.Println("Successfully updated codelist -> ",mgr.codelist)
					//fmt.Println(response)
					fmt.Println(mgr.codelist, " updated.")
				}
			}
		}
		if len(codelistErrors) > 0 {
			fmt.Printf("Errors found for CodeList %s\n", mgr.codelist)
			for _, errormsg := range codelistErrors {
				fmt.Printf("%s\n", errormsg)
			}
		}
	}
	return nil
	//fmt.Println(clist)
}

func decrypt(text string) string {
	if text == "" {
		return ""
	}
	decrypted, err := amf_crypto.Decrypt(text)
	if err != nil {
		return ""
	}
	return decrypted
}

func (mgr *apiMgr) BulkUpdate(payload string) (string, error) {
	codelistid, err := mgr.GetCodelistID()
	if err != nil {
		fmt.Println("Failed to get code list item", err)
		return "", fmt.Errorf("Code list not found")
	}
	if len(codelistid) == 0 {
		return "", fmt.Errorf("Codelist not found")
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", mgr.apiurl+"/B2BAPIs/svc/codelists/"+codelistid+"/actions/bulkupdatecodes", bytes.NewBuffer([]byte(payload)))
	req.SetBasicAuth(mgr.username, mgr.password)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ERROR - API call failed [%s]", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ERROR - Invalid API response [%s]", err)
	}
	if 200 != resp.StatusCode {
		return "", fmt.Errorf("ERROR - Invalid API response for Code List Bulk Update API call [%s]", body)
	}
	return string(body), nil
}

func (mgr *apiMgr) CreateCodelist(payload string) (string, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req, err := http.NewRequest("POST", mgr.apiurl+"/B2BAPIs/svc/codelists/", bytes.NewBuffer([]byte(payload)))
	req.SetBasicAuth(mgr.username, mgr.password)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ERROR - Create Code List API call failed [%s]", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ERROR - Invalid API response for Create Code List API call [%s]", err)
	}
	if 201 != resp.StatusCode {
		fmt.Println("Response Code", resp.StatusCode)
		return "", fmt.Errorf("%s", body)
	}
	return string(body), nil
}

func (mgr *apiMgr) GetCodelistID() (string, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	queryParams := "?locale=en_US&codeListName=" + mgr.codelist + "&_accept=application/json&_contentType=application/json&_exclude=codes"
	req, err := http.NewRequest("GET", mgr.apiurl+"/B2BAPIs/svc/codelists/"+queryParams, nil)
	req.SetBasicAuth(mgr.username, mgr.password)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ERROR - Read Code List API call failed [%s]", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ERROR - Invalid API response for Read Code List API Call [%s]", err)
	}
	if 200 != resp.StatusCode {
		return "", fmt.Errorf("%s", body)
	}
	var codelist interface{}
	//fmt.Println("body",string(body))
	err = json.Unmarshal([]byte(string(body)), &codelist)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	data := codelist.([]interface{})
	//fmt.Println(data[0])
	for _, value := range data {
		data2 := value.(map[string]interface{})
		for k, v := range data2 {
			switch v := v.(type) {
			case string:
				if k == "_id" {
					return v, nil
				}
			}
		}
	}

	return "", nil
}

func (mgr *apiMgr) backupCodelist() error {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	queryParams := "?locale=en_US&codeListName=" + mgr.codelist + "&_accept=application/json&_contentType=application/json&_exclude=codes"
	req, err := http.NewRequest("GET", mgr.apiurl+"/B2BAPIs/svc/codelists/"+queryParams, nil)
	req.SetBasicAuth(mgr.username, mgr.password)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if 200 != resp.StatusCode {
		return fmt.Errorf("ERROR - Invalid API response for Read Code List API call [%s]", body)
	}
	var codelist interface{}
	//fmt.Println("body",string(body))
	err = json.Unmarshal([]byte(string(body)), &codelist)
	if err != nil {
		fmt.Println(err)
		return err
	}
	data := codelist.([]interface{})
	//fmt.Println(data[0])
	for _, value := range data {
		data2 := value.(map[string]interface{})
		codelist2 := &CodeListItem{}
		codelist2.codeListName = mgr.codelist
		codelist2.codes = make([]Code, 0)
		for k, v := range data2 {

			switch v := v.(type) {
			case string:
				if k == "_id" {
					codelist2._id = v
				}
				//fmt.Println(k, v, "(string)")
			case float64:
				//fmt.Println(k, v, "(float64)")
				if k == "listStatus" {
					codelist2.listStatus = v
				} else if k == "versionNumber" {
					codelist2.versionNumber = v
				}
			case []interface{}:
				//fmt.Println(k, "(array):")

				for _, u := range v {
					codeitem, err := mgr.getCodeFromInterface(u)
					if err != nil {
						fmt.Println(err)
					} else if codeitem != nil {
						codelist2.codes = append(codelist2.codes, *codeitem)
						//fmt.Println("Got CODE",codeitem)
					}
				}
			default:
				fmt.Println(k, v, "(unknown)")
			}
		}
		//mgr.showCodeListItem(*codelist2)
		mgr.WriteCodeListItem(*codelist2)

	}

	return nil
}

func (mgr *apiMgr) WriteCodeListItem(codelist CodeListItem) {
	//sheetname:=codelist.codeListName+"#"+strconv.Itoa(int(codelist.versionNumber))
	sheetname := codelist._id
	mgr.bkpfileptr.NewSheet(sheetname)
	mgr.bkpfileptr.SetCellValue(sheetname, "A1", "Action")
	mgr.bkpfileptr.SetCellValue(sheetname, "B1", "SenderCode")
	mgr.bkpfileptr.SetCellValue(sheetname, "C1", "ReceiverCode")
	mgr.bkpfileptr.SetCellValue(sheetname, "D1", "Description")
	mgr.bkpfileptr.SetCellValue(sheetname, "E1", "Text1")
	mgr.bkpfileptr.SetCellValue(sheetname, "F1", "Text2")
	mgr.bkpfileptr.SetCellValue(sheetname, "G1", "Text3")
	mgr.bkpfileptr.SetCellValue(sheetname, "H1", "Text4")
	mgr.bkpfileptr.SetCellValue(sheetname, "I1", "Text5")
	mgr.bkpfileptr.SetCellValue(sheetname, "J1", "Text6")
	mgr.bkpfileptr.SetCellValue(sheetname, "K1", "Text7")
	mgr.bkpfileptr.SetCellValue(sheetname, "L1", "Text8")
	mgr.bkpfileptr.SetCellValue(sheetname, "M1", "Text9")
	for i := 0; i < len(codelist.codes); i++ {
		mgr.bkpfileptr.SetCellValue(sheetname, "A"+strconv.Itoa(i+2), "Yes")
		mgr.bkpfileptr.SetCellValue(sheetname, "B"+strconv.Itoa(i+2), codelist.codes[i].senderCode)
		mgr.bkpfileptr.SetCellValue(sheetname, "C"+strconv.Itoa(i+2), codelist.codes[i].receiverCode)
		mgr.bkpfileptr.SetCellValue(sheetname, "D"+strconv.Itoa(i+2), codelist.codes[i].description)
		mgr.bkpfileptr.SetCellValue(sheetname, "E"+strconv.Itoa(i+2), codelist.codes[i].text1)
		mgr.bkpfileptr.SetCellValue(sheetname, "F"+strconv.Itoa(i+2), codelist.codes[i].text2)
		mgr.bkpfileptr.SetCellValue(sheetname, "G"+strconv.Itoa(i+2), codelist.codes[i].text3)
		mgr.bkpfileptr.SetCellValue(sheetname, "H"+strconv.Itoa(i+2), codelist.codes[i].text4)
		mgr.bkpfileptr.SetCellValue(sheetname, "I"+strconv.Itoa(i+2), codelist.codes[i].text5)
		mgr.bkpfileptr.SetCellValue(sheetname, "J"+strconv.Itoa(i+2), codelist.codes[i].text6)
		mgr.bkpfileptr.SetCellValue(sheetname, "K"+strconv.Itoa(i+2), codelist.codes[i].text7)
		mgr.bkpfileptr.SetCellValue(sheetname, "L"+strconv.Itoa(i+2), codelist.codes[i].text8)
		mgr.bkpfileptr.SetCellValue(sheetname, "M"+strconv.Itoa(i+2), codelist.codes[i].text9)
	}
}

func (mgr *apiMgr) showCodeListItem(codelist CodeListItem) {
	fmt.Println("Code List Name ", codelist.codeListName)
	fmt.Println("List Status", codelist.listStatus)
	fmt.Println("Version Number", codelist.versionNumber)

	for i := 0; i < len(codelist.codes); i++ {
		fmt.Println("Sender Code ", codelist.codes[i].senderCode)
		fmt.Println("Receiver Code ", codelist.codes[i].receiverCode)
		fmt.Println("Description ", codelist.codes[i].description)
		for j := 1; j < 9; j++ {
			switch j {
			case 1:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text1)
			case 2:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text2)
			case 3:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text3)
			case 4:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text4)
			case 5:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text5)
			case 6:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text6)
			case 7:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text7)
			case 8:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text8)
			case 9:
				fmt.Println("Text"+strconv.Itoa(j), codelist.codes[i].text9)
			}

		}
	}
}
func (mgr *apiMgr) getCodeFromInterface(in interface{}) (*Code, error) {
	code := &Code{}
	code.senderCode = mgr.getCodeField(in, "senderCode")
	code.receiverCode = mgr.getCodeField(in, "receiverCode")
	code.description = mgr.getCodeField(in, "description")
	code.text1 = mgr.getCodeField(in, "text1")
	code.text2 = mgr.getCodeField(in, "text2")
	code.text3 = mgr.getCodeField(in, "text3")
	code.text4 = mgr.getCodeField(in, "text4")
	code.text5 = mgr.getCodeField(in, "text5")
	code.text6 = mgr.getCodeField(in, "text6")
	code.text7 = mgr.getCodeField(in, "text7")
	code.text8 = mgr.getCodeField(in, "text8")
	code.text9 = mgr.getCodeField(in, "text9")
	return code, nil
}

func (mgr *apiMgr) getCodeField(in interface{}, key string) string {
	obj, ok := in.(map[string]interface{})
	if ok {
		if val, ok := obj[key]; ok {
			return mgr.getString(val)
		}
	}
	return ""
}

func (mgr *apiMgr) getString(in interface{}) string {
	if str, ok := in.(string); ok {
		return str
	}
	return ""
}

func (mgr *apiMgr) deleteCodelist(_id string) error {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	queryParams := _id + "?locale=en_US&codeListName=" + mgr.codelist + "&_accept=application/json&_contentType=application/json&_exclude=codes"
	req, err := http.NewRequest("DELETE", mgr.apiurl+"/B2BAPIs/svc/codelists/"+queryParams, nil)
	req.SetBasicAuth(mgr.username, mgr.password)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ERROR - Invalid API response for Delete Code List API call [%s]", err)
	}
	if 200 != resp.StatusCode {
		return fmt.Errorf("ERROR - Invalid API response for Delete Code List API call [%s]", body)
	}
	return nil
}

func (mgr *apiMgr) codelistFailed(codelistArr []string) bool {
	for _, st := range codelistArr {
		if strings.Contains(st, mgr.codelist) {
			return true
		}
	}
	return false
}
