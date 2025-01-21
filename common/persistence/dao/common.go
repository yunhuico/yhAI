package dao

import (
	"errors"
	"strings"
	"time"

	"linkernetworks.com/dcos-backend/common/persistence/session"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var DAO *Dao

const ParamID = "_id" // mongo id parameter

type (
	QueryStruct struct {
		CollectionName string
		Selector       bson.M
		Skip           int
		Limit          int
		Sort           string
	}
	Dao struct {
		SessMng    *session.SessionManager
		MongoAlias string
	}
)

func (d *Dao) ParamID() string {
	return ParamID
}

func remove_dots(doc interface{}) (newDoc bson.M) {
	// get document type
	document := bson.M{}
	in, _ := bson.Marshal(doc)
	bson.Unmarshal(in, &document)

	newDoc = bson.M{}
	// logrus.Debugf("remove dot from document %v", document)
	for key, subdoc := range document {
		if subdoc, ok := subdoc.(bson.M); ok {
			newDoc[key] = remove_dots(subdoc)
		} else {
			newDoc[key] = document[key]
		}

		if strings.Contains(key, ".") {
			logrus.Infof("replace dot for key %v", key)
			newDoc[strings.Replace(key, ".", "\u2024", -1)] = newDoc[key]
			delete(newDoc, key)
		}
	}
	return
}

//get current time as string formatted to RFC3339
//e.g 2015-12-28T15:53:28+08:00
func GetCurrentTime() (t time.Time) {
	t, err := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	if err != nil {
		logrus.Errorln("error parse time, %v", err)
	}
	return
}

func HandleInsert(collName string, document interface{}) (err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}
	// Close session if it's needed
	if needclose {
		defer session.Close()
	}

	// Mongo Collection
	col := getCollection(collName, session)

	// Insert document to collection
	if err = col.Insert(document); err != nil {
		logrus.Errorf("Error inserting document: %v", err)
		return
	}
	return
}

func HandleQueryOne(document interface{}, queryStruct QueryStruct) (err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}
	// Close session if it's needed
	if needclose {
		defer session.Close()
	}
	// Compose a query from request
	query, err := composeQuery(session, queryStruct, true)
	if err != nil {
		return
	}
	// Get one document
	err = query.One(document)
	return
}

func HandleQueryAll(documents interface{}, queryStruct QueryStruct) (total int, err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}
	// Close session if it's needed
	if needclose {
		defer session.Close()
	}

	// Compose a query from request
	query, err := composeQuery(session, queryStruct, false)
	if err != nil {
		return
	}

	// Get all documents
	if err = query.All(documents); err != nil {
		return
	}

	// Count documents if count parameter is included in query
	query.Skip(0)
	query.Limit(0)
	if n, err := query.Count(); err == nil {
		total = n
	}
	return
}

func HandleUpdateOne(document interface{}, queryStruct QueryStruct) (created bool, err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}

	// Close session if it's needed
	if needclose {
		defer session.Close()
	}

	// Mongo Collection
	col := getCollection(queryStruct.CollectionName, session)

	// Update document(/s)
	// replace dot in field name to /u2024
	// FIXME: replace
	// st := reflect.TypeOf(document)
	newbson := remove_dots(document)
	in, _ := bson.Marshal(newbson)
	bson.Unmarshal(in, document)

	var (
		info *mgo.ChangeInfo
	)

	// Update document by id
	info, err = col.UpsertId(queryStruct.Selector[ParamID], document)
	if err != nil {
		return
	}

	// Get updated id from mongo
	if info != nil && info.UpsertedId != nil {
		// docid, _ = info.UpsertedId.(string)
		created = (info.Updated == 0)
	}

	return
}

func HandleDelete(collName string, one bool, selector bson.M) (err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}

	// Close session if it's needed
	if needclose {
		defer session.Close()
	}

	// Mongo Collection
	col := getCollection(collName, session)

	if len(selector) == 0 {
		err = errors.New("can not drop entire collection, selector can not be empty")
		return
	}

	// Remove one document if no query, otherwise remove all matching query
	if one {
		err = col.Remove(selector)
	} else {
		_, err = col.RemoveAll(selector)
	}

	return
}

func getCollection(collName string, session *mgo.Session) *mgo.Collection {
	config, err := DAO.SessMng.GetConfig(DAO.MongoAlias)
	if err != nil {
		logrus.Errorf("get db config error %+v, use default database ", err)
		return session.DB("").C(collName)
	}
	db := config.GetString("database", "")
	return session.DB(db).C(collName)
}

func composeQuery(session *mgo.Session, queryStruct QueryStruct, one bool) (query *mgo.Query, err error) {
	// logrus.Debugf("compose query with params: selector=%v, one=%v, fields=%v, skip=%d, limit=%d, sort=%s", selector, one, fields, skip, limit, sort)
	// Mongo Collection
	// logrus.Debugf("composeQuery is called with parameters [queryStruct=%v]", queryStruct)

	col := getCollection(queryStruct.CollectionName, session)

	// Create a Mongo Query
	query = col.Find(queryStruct.Selector)
	// // Fields of document to select
	// if len(fields) > 0 {
	// 	query.Select(fields)
	// }

	// If selects one from `_id` parameter that's all
	if one {
		return
	}

	// Number of documents to skip in result set
	query.Skip(queryStruct.Skip)

	// Maximum number of documents in the result set
	query.Limit(queryStruct.Limit)

	// Compose sort from comma separated list in request query
	if len(queryStruct.Sort) > 0 {
		query.Sort(strings.Split(queryStruct.Sort, ",")...)
	}

	return
}

func HandleUpdateByQueryPartial(collName string, selector bson.M, document interface{}) (err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}

	// Close session if it's needed
	if needclose {
		defer session.Close()
	}

	// Mongo Collection
	col := getCollection(collName, session)

	//Set time_update
	change := bson.M{"$set": document}
	_, err = col.UpdateAll(selector, change)
	return
}

func HandlePartialUpdateByQuery(collName string, selector bson.M, document bson.M) (err error) {
	// Mongo session
	session, needclose, err := DAO.SessMng.GetDefault()
	if err != nil {
		return
	}

	// Close session if it's needed
	if needclose {
		defer session.Close()
	}

	// Mongo Collection
	col := getCollection(collName, session)

	change := bson.M{"$set": document}
	err = col.Update(selector, change)

	return
}
