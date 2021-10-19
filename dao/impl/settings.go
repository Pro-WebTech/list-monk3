package impl

import (
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
	"log"
	"strings"
)

func NewSettingsDaoImpl(lo *log.Logger) *SettingsDaoImpl {
	return &SettingsDaoImpl{lo: lo}
}

type SettingsDaoImpl struct {
	lo *log.Logger
}

func (u *SettingsDaoImpl) FindAStats(db *sqlx.DB) (settingJs types.JSONText, err error) {
	var buf strings.Builder
	buf.WriteString("SELECT JSON_BUILD_OBJECT('subscribers', JSON_BUILD_OBJECT('total', (SELECT COUNT(*) FROM subscribers), ")
	buf.WriteString("'blocklisted', (SELECT COUNT(*) FROM subscribers WHERE status='blocklisted'),'orphans', (SELECT COUNT(id) ")
	buf.WriteString("FROM subscribers LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id) ")
	buf.WriteString("WHERE subscriber_lists.subscriber_id IS NULL)), 'lists', JSON_BUILD_OBJECT('total', (SELECT COUNT(*) FROM lists), ")
	buf.WriteString("'private', (SELECT COUNT(*) FROM lists WHERE type='private'),'fitlered', (select COUNT(*) from lists where name like 'FLTRD-%'), ")
	buf.WriteString("'public', (SELECT COUNT(*) FROM lists WHERE type='public'), 'optin_single', (SELECT COUNT(*) FROM lists WHERE optin='single'), ")
	buf.WriteString("'optin_double', (SELECT COUNT(*) FROM lists WHERE optin='double')), 'campaigns', JSON_BUILD_OBJECT('total', (SELECT COUNT(*) FROM campaigns), ")
	buf.WriteString("'finished', (SELECT COUNT(*) AS num FROM campaigns where status = 'finished'), 'cancelled', (SELECT COUNT(*) AS num ")
	buf.WriteString("FROM campaigns where status = 'cancelled'), 'draft', (SELECT COUNT(*) AS num FROM campaigns where status = 'draft')), ")
	buf.WriteString("'messages',  JSON_BUILD_OBJECT('total', (SELECT value AS messages FROM settings where key = 'emailsent.total'), ")
	buf.WriteString("'bounces', (select COUNT(*)  from events s where  event_type =  'Bounced' and event_timestamp > NOW() - INTERVAL '24 HOURS'), ")
	buf.WriteString("'bounces2h', (select COUNT(*)  from events s where  event_type =  'Bounced' and event_timestamp > NOW() - INTERVAL '2 HOURS'), ")
	buf.WriteString("'complaints', (select COUNT(*)  from events s where  event_type = 'Complained' and event_timestamp > NOW() - INTERVAL '24 HOURS'), ")
	buf.WriteString("'complaints2h', (select COUNT(*)  from events s where  event_type = 'Complained' and event_timestamp > NOW() - INTERVAL '2 HOURS'), ")
	buf.WriteString("'unsubscribed', (select COUNT(*)  from events s where  event_type = 'unsubscribed' and event_timestamp > NOW() - INTERVAL '24 HOURS'), ")
	buf.WriteString("'unsubscribed2h', (select COUNT(*)  from events s where  event_type = 'unsubscribed' and event_timestamp > NOW() - INTERVAL '2 HOURS')), ")
	buf.WriteString("'performance',  JSON_BUILD_OBJECT('concurrency', (SELECT value AS messages FROM settings where key = 'app.concurrency'), ")
	buf.WriteString("'message_rate', (SELECT value AS messages FROM settings where key = 'app.message_rate'), ")
	buf.WriteString("'batch_size', (SELECT value AS messages FROM settings where key = 'app.batch_size'), ")
	buf.WriteString("'max_error_threshold', (SELECT value AS messages FROM settings where key = 'app.max_send_errors'), ")
	buf.WriteString("'sliding_window_limit', (SELECT value AS messages FROM settings where key = 'app.message_sliding_window'), ")
	buf.WriteString("'sliding_max_messages', (SELECT value AS messages FROM settings where key = 'app.message_sliding_window_rate'), ")
	buf.WriteString("'sliding_duration', (SELECT value AS messages FROM settings where key = 'app.message_sliding_window_duration'))) as settingJs")

	rows, err := db.Query(buf.String())
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		u.lo.Printf("err FindAll[SettingsDaoImpl]: %v", err)
		return
	}

	for rows.Next() {
		err = rows.Scan(&settingJs)
	}

	return
}

func (u *SettingsDaoImpl) FindAll(db *sqlx.DB) (entity []models.Settings, err error) {
	entity = []models.Settings{}
	var buf strings.Builder
	buf.WriteString("SELECT key, value from ")
	buf.WriteString(models.TblSettings)
	buf.WriteString(" WHERE 1 = $1 ")
	bind := []interface{}{1}

	rows, err := db.Query(buf.String(), bind...)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		u.lo.Printf("err FindAll[SettingsDaoImpl]: %v", err)
		return entity, err
	}

	for rows.Next() {
		var eachRow models.Settings
		//structs.MergeRow(rows, &eachRow)
		rows.Scan(&eachRow.Key, &eachRow.Value)
		entity = append(entity, eachRow)
	}

	return entity, nil
}

func (u *SettingsDaoImpl) UpdateValue(db *sqlx.DB, key string, value string) error {
	var buf strings.Builder
	buf.WriteString("update ")
	buf.WriteString(models.TblSettings)
	buf.WriteString(" SET value = $1 ")
	buf.WriteString(" WHERE key = $2 ")
	_, err := db.Exec(buf.String(), value, key)
	if err != nil {
		u.lo.Printf("err FindAll[SettingsDaoImpl]: %v", err)
		return err
	}

	return nil
}

func (u *SettingsDaoImpl) FindByKey(db *sqlx.DB, key string) (entity models.Settings, err error) {
	entity = models.Settings{}
	var buf strings.Builder
	buf.WriteString("SELECT key, value from ")
	buf.WriteString(models.TblSettings)
	buf.WriteString(" WHERE 1 = $1 AND key = $2 ")
	bind := []interface{}{1, key}

	rows, err := db.Query(buf.String(), bind...)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		u.lo.Printf("err FindByKey[SettingsDaoImpl]: %v", err)
		return entity, err
	}

	for rows.Next() {
		rows.Scan(&entity.Key, &entity.Value)
	}

	return entity, nil
}
