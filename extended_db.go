package extendedgorm

import (
	"fmt"
	"log"
	"reflect"

	"github.com/tmdgo/environment/variables"
	"github.com/tmdgo/reflection/fields"
	"github.com/tmdgo/reflection/interfaces"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ExtendedDB struct {
	GormDB         *gorm.DB
	connectionName string
}

func (ExtendedDB) log(message string) {
	log.Printf("extendedgorm: extendeddb: %v\n", message)
}

func (ExtendedDB) panic(message string, err error) {
	log.Panicf("extendedgorm: extendeddb: %v - error: %v", message, err.Error())
}

func (ExtendedDB) panicf(format string, a ...interface{}) {
	log.Panicf("extendedgorm: extendeddb: %v\n", fmt.Sprintf(format, a...))
}

func (ExtendedDB) error(err error) error {
	return fmt.Errorf("extendedgorm: extendeddb: %v", err.Error())
}

func (ExtendedDB) errorf(format string, a ...interface{}) (err error) {
	err = fmt.Errorf("extendedgorm: extendeddb: %v", fmt.Sprintf(format, a...))
	return
}

func (extendedDB *ExtendedDB) connectionTest() (err error) {
	goDB, err := extendedDB.GormDB.DB()
	if err != nil {
		extendedDB.panic("unable to connect to database please verify the extendeddb enviroment variables and database configuration for this application", err)
	}
	err = goDB.Ping()
	if err != nil {
		extendedDB.panic("unable to connect to database please verify the extendeddb enviroment variables and database configuration for this application", err)
	}
	err = extendedDB.GormDB.Exec("SELECT 'value1' as field1, 'value2' as field2").Error
	if err != nil {
		extendedDB.panic("unable to connect to database please verify the extendeddb enviroment variables and database configuration for this application", err)
	}
	return
}

func (extendedDB *ExtendedDB) Connect(name string) (err error) {
	extendedDB.connectionName = name

	getDsnFromEnvironment := func() (extendedDBType, dsn string) {
		envVarPattern := "EXTENDEDDB_" + extendedDB.connectionName + "_%v"

		errorsCount := 0

		getParameterFromEnvironment := func(name string) (value string) {
			envVarName := fmt.Sprintf(envVarPattern, name)
			value, err := variables.Get(envVarName)
			if err != nil {
				errorsCount++
				log.Println(err)
			} else if value == "" {
				errorsCount++
				extendedDB.log(fmt.Sprintf("please set the %v environment variable\n", envVarName))
			}
			return
		}

		getIntParameterFromEnvironment := func(name string) (value int64) {
			envVarName := fmt.Sprintf(envVarPattern, name)
			value, _ = variables.GetInt64(envVarName)
			if err != nil {
				errorsCount++
				log.Println(err)
			} else if value == 0 {
				errorsCount++
				extendedDB.log(fmt.Sprintf("please set the %v environment variable\n", envVarName))
			}
			return
		}

		extendedDBType = getParameterFromEnvironment("TYPE")
		host := getParameterFromEnvironment("HOST")
		port := getIntParameterFromEnvironment("PORT")
		sslMode := getParameterFromEnvironment("SSL_MODE")
		extendedDBName := getParameterFromEnvironment("NAME")
		user := getParameterFromEnvironment("USER")
		password := getParameterFromEnvironment("PASSWORD")

		if errorsCount != 0 {
			extendedDB.panicf("unable to get all information to connect to extendedDB please read previous log")
		}

		dsn = fmt.Sprintf(
			"host=%v port=%v sslmode=%v dbname=%v user=%v password=%v",
			host,
			port,
			sslMode,
			extendedDBName,
			user,
			password,
		)

		return
	}

	_, dsn := getDsnFromEnvironment()

	extendedDB.GormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		extendedDB.panic("unable to connect to database please verify the extendeddb enviroment variables and database configuration for this application", err)
	}

	err = extendedDB.connectionTest()
	if err != nil {
		return
	}

	return
}

func (extendedDB *ExtendedDB) Create(model interface{}) (err error) {
	id, err := extendedDB.getEntityID(model)
	if err != nil {
		return
	}
	if id != 0 {
		err = extendedDB.errorf(
			`it is not possible to insert a model "%v" with the pre-filled ID field`,
			interfaces.GetTypeName(model),
		)
		return
	}
	err = extendedDB.GormDB.Create(model).Error
	if err != nil {
		err = extendedDB.errorf("%s", err)
		return
	}
	return
}

func (extendedDB *ExtendedDB) Update(model interface{}) (err error) {
	id, err := extendedDB.getEntityID(model)
	if err != nil {
		return
	}
	if id == 0 {
		err = extendedDB.errorf(
			`it is not possible to update a model "%v" with the blank ID field`,
			interfaces.GetTypeName(model),
		)
		return
	}
	err = extendedDB.GormDB.Save(model).Error
	if err != nil {
		err = extendedDB.error(err)
	}
	return
}

func (extendedDB *ExtendedDB) DeleteByID(model interface{}, id uint) (err error) {
	err = extendedDB.GormDB.Delete(model, id).Error
	if err != nil {
		err = extendedDB.error(err)
	}
	return
}

func (extendedDB *ExtendedDB) SelectByID(model interface{}, id uint) (err error) {
	err = extendedDB.GormDB.First(model, 10).Error
	if err != nil {
		err = extendedDB.error(err)
	}
	return
}

func (extendedDB *ExtendedDB) SelectAll(models interface{}) (err error) {
	err = extendedDB.GormDB.Find(models).Error
	if err != nil {
		if err.Error() == "record not found" {
			err = nil
			return
		}
		err = extendedDB.error(err)
	}
	return
}

func (extendedDB *ExtendedDB) Filter(models, modelFilter interface{}) (err error) {
	err = extendedDB.GormDB.Where(modelFilter).Find(models).Error
	if err != nil {
		if err.Error() == "record not found" {
			err = nil
			return
		}
		err = extendedDB.error(err)
	}
	return
}

func (extendedDB *ExtendedDB) Transaction(function func(extendedDB ExtendedDB) error) (err error) {
	tx := extendedDB.GormDB.Begin()
	txDatabase := ExtendedDB{GormDB: tx, connectionName: extendedDB.connectionName}
	err = function(txDatabase)
	if err != nil {
		txDatabase.GormDB.Rollback()
		return
	}
	txDatabase.GormDB.Commit()
	return
}

func (extendedDB *ExtendedDB) RegisterEntities(entities ...interface{}) {
	extendedDB.GormDB.AutoMigrate(entities...)
}

func (extendedDB *ExtendedDB) getEntityID(entity interface{}) (id uint, err error) {
	fieldType, fieldValue, err := fields.GetTypeAndValue(entity, "ID")
	if err != nil {
		return
	}
	if reflect.TypeOf(uint(0)) != fieldType {
		err = extendedDB.errorf(`the "%v" entity ID field is not of type uint`, interfaces.GetTypeName(entity))
	}
	id = fieldValue.(uint)
	return
}
