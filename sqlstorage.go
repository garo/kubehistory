package main

import (
	"github.com/golang-migrate/migrate/v4"
	"database/sql"
	"github.com/golang/glog"
	"github.com/lib/pq"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	api "k8s.io/api/core/v1"
	"encoding/json"
)

type SnapshotReason string
const REASON_CREATED SnapshotReason = "c"
const REASON_UPDATED SnapshotReason = "u"
const REASON_DELETED SnapshotReason = "d"
const REASON_SNAPSHOT SnapshotReason = "s"

type PersistentStorage interface {
	GetDB() (*sql.DB)
	StorePod(reason SnapshotReason, pod *api.Pod) (error)
	StoreNode(reason SnapshotReason, node *api.Node) (error)
	Cleanup() (error)
}

type SqlStorage struct {
	db *sql.DB
	driver database.Driver
	uri string
}

func NewSqlStorage(uri string) (*SqlStorage, error) {
	s := SqlStorage{
		uri: uri,
	}
	return &s, nil
}

func (s *SqlStorage) GetDB() (*sql.DB) {
	return s.db
}

func (s *SqlStorage) Init() (error) {
	glog.Infof("Opening SQL connection to %s", s.uri)
	db, err := sql.Open("postgres", s.uri)
	if err != nil {
		panic(err)
		return err
	}

	db.SetMaxOpenConns(20)

	s.db = db

	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
	if err != nil {
		panic(err)
		return err
	}
	s.driver = driver

	glog.Infof("Running SQL migrations...")
	m, err := migrate.NewWithDatabaseInstance("file://migrations",	"postgres", driver)
	if err != nil {
		panic(err)
		return err
	}

	m.Migrate(1)

	return nil
}

func (s *SqlStorage) Cleanup() (error) {
	s.db.Query("DELETE FROM pods WHERE ts < NOW() - INTERVAL '1 week'")
	s.db.Query("DELETE FROM nodes WHERE ts < NOW() - INTERVAL '2 weeks'")
	return nil
}


func (s *SqlStorage) StorePod(reason SnapshotReason, pod *api.Pod) (error) {

	str, _ := json.Marshal(pod)

	statement := `
	INSERT INTO pods
		(reason, name, namespace, selfLink, uid, resourceVersion, creationTimestamp, deletionTimestamp,
			nodeName, hostIP, podIP, data)
	VALUES
	  (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12
		)
`

	creationTimestamp := pq.NullTime{
		Time: pod.ObjectMeta.CreationTimestamp.Time,
		Valid: !pod.ObjectMeta.CreationTimestamp.Time.IsZero(),
	}

	deletionTimestamp := pq.NullTime{
		Valid: false,
	}
	if pod.ObjectMeta.DeletionTimestamp != nil {
		deletionTimestamp.Time = pod.ObjectMeta.DeletionTimestamp.Time
		deletionTimestamp.Valid = true
	}

	_, err := s.db.Exec(statement, reason, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, pod.ObjectMeta.SelfLink,
		pod.ObjectMeta.UID, pod.ObjectMeta.ResourceVersion, creationTimestamp, deletionTimestamp,
		pod.Spec.NodeName,
		pod.Status.HostIP, pod.Status.PodIP,
	  str)

	if err != nil {
		glog.Infof("Error doing sql update: %+v", err)
	}

	return nil
}


func (s *SqlStorage) StoreNode(reason SnapshotReason, node *api.Node) (error) {

	str, _ := json.Marshal(node)

	statement := `
	INSERT INTO nodes
		(reason, name, selfLink, uid, resourceVersion, creationTimestamp, deletionTimestamp,
			hostIP, data)
	VALUES
	  (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9
		)
`

	creationTimestamp := pq.NullTime{
		Time: node.ObjectMeta.CreationTimestamp.Time,
		Valid: !node.ObjectMeta.CreationTimestamp.Time.IsZero(),
	}

	deletionTimestamp := pq.NullTime{
		Valid: false,
	}
	if node.ObjectMeta.DeletionTimestamp != nil {
		deletionTimestamp.Time = node.ObjectMeta.DeletionTimestamp.Time
		deletionTimestamp.Valid = true
	}

	hostip := ""
	for _, na := range node.Status.Addresses {
		if na.Type == api.NodeInternalIP {
			hostip = na.Address
		}
	}

	_, err := s.db.Exec(statement, reason, node.ObjectMeta.Name, node.ObjectMeta.SelfLink,
		node.ObjectMeta.UID, node.ObjectMeta.ResourceVersion, creationTimestamp, deletionTimestamp,
		hostip,
	  str)

	if err != nil {
		glog.Infof("Error doing sql update: %+v", err)
	}

	return nil
}
