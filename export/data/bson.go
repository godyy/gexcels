package data

import (
	"context"
	"errors"
	"time"

	"github.com/godyy/gexcels"
	"github.com/godyy/gexcels/export"
	internal_define "github.com/godyy/gexcels/export"
	"github.com/godyy/gexcels/internal/log"
	"github.com/godyy/gexcels/parse"
	pkg_errors "github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ErrNoMongoDBSpecified 没有指定MongoDB数据库
var ErrNoMongoDBSpecified = errors.New("no mongo db specified")

const (
	bsonSaveBatchSize = 1000             // 每次插入的文档数量
	bsonSaveTimeout   = 10 * time.Second // 每次插入超时时间
)

// createBsonSaveContext 创建bson导出上下文
func createBsonSaveContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), bsonSaveTimeout)
}

// ExportBson 导出bson格式数据
func ExportBson(p *parse.Parser, db *mongo.Database) error {
	if p == nil {
		return ErrNoParserSpecified
	}
	if db == nil {
		return ErrNoMongoDBSpecified
	}
	return doExport(&bsonExporter{parser: p, db: db})
}

// bsonExporter bson导出器
type bsonExporter struct {
	parser *parse.Parser
	db     *mongo.Database
}

func (e *bsonExporter) kind() internal_define.DataKind {
	return internal_define.DataBson
}

func (e *bsonExporter) export() error {
	log.Printf("export data bson to [%s]", e.db.Name())

	for _, table := range e.parser.Tables {
		if err := e.exportTable(table); err != nil {
			return pkg_errors.WithMessage(err, "export data: bson")
		}
		log.PrintfGreen("export data: bson: table[%s] to [%s]", table.Name, e.db.Name())
	}

	return nil
}

func (e *bsonExporter) exportTable(table *parse.Table) error {
	if table.IsGlobal {
		return e.exportGlobalTable(table)
	} else {
		return e.exportNormalTable(table)
	}
}

func (e *bsonExporter) exportNormalTable(table *parse.Table) error {
	var (
		coll         = e.db.Collection(table.Name)
		batchObjects = make([]bson.D, 0, bsonSaveBatchSize)
	)

	for i, entry := range table.Entries {
		object := make(bson.D, 0, len(table.Fields))
		for _, fd := range table.Fields {
			if fd.Col == gexcels.TableColFieldID {
				object = append(object, bson.E{Key: export.TableFieldIDBsonName, Value: entry[fd.Name]})
			} else {
				object = append(object, bson.E{Key: fd.Name, Value: entry[fd.Name]})
			}
		}

		batchObjects = append(batchObjects, object)
		if len(batchObjects) >= bsonSaveBatchSize || i == len(table.Entries)-1 {
			ctx, cancel := createBsonSaveContext()

			if _, err := coll.InsertMany(ctx, batchObjects); err != nil {
				cancel()
				return pkg_errors.WithMessagef(err, "table[%s] insert failed at:%d batch:%d", table.Name, i-len(batchObjects), len(batchObjects))
			}

			cancel()
			batchObjects = batchObjects[:0]
		}
	}

	return nil
}

func (e *bsonExporter) exportGlobalTable(td *parse.Table) error {
	var (
		coll   = e.db.Collection(export.TableGlobalBsonCollName)
		object = make(bson.D, 0, len(td.Fields))
	)

	object = append(object, bson.E{Key: export.TableFieldIDBsonName, Value: td.Name})
	for _, fd := range td.Fields {
		object = append(object, bson.E{Key: fd.Name, Value: td.GetEntryByName(fd.Name)})
	}

	ctx, cancel := createBsonSaveContext()
	defer cancel()

	if _, err := coll.InsertOne(ctx, object); err != nil {
		return pkg_errors.WithMessagef(err, "table[%s] insert to [%s] failed", td.Name, export.TableGlobalBsonCollName)
	}

	return nil
}
