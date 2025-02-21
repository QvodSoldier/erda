// Copyright (c) 2021 Terminus, Inc.
//
// This program is free software: you can use, redistribute, and/or modify
// it under the terms of the GNU Affero General Public License, version 3
// or later ("AGPL"), as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package xmind

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"

	"github.com/erda-project/erda/pkg/filehelper"
)

const (
	// META-INF/manifest.xml 使用固定值即可
	fixedManifestXmlFileContent = `<?xml version="1.0" encoding="UTF-8" standalone="no"?> <manifest xmlns="urn:xmind:xmap:xmlns:manifest:1.0"> <file-entry full-path="content.xml" media-type="text/xml"/> <file-entry full-path="META-INF/" media-type=""/> <file-entry full-path="META-INF/manifest.xml" media-type="text/xml"/> <file-entry full-path="meta.xml" media-type="text/xml"/> </manifest>`
	// meta.xml 使用固定值即可
	fixedMetaXmlFileContent = `<?xml version="1.0" encoding="UTF-8" standalone="no"?> <meta xmlns="urn:xmind:xmap:xmlns:meta:2.0" version="2.0"> <Author> <Name>root</Name> </Author> <Create> <Time>Aug 26, 2020 10:29:03 AM</Time> </Create> </meta>`
)

// XMLTopicType topics 类型，必填属性(attr)
type XMLTopicType string

var (
	TopicsTypeAttached XMLTopicType = "attached" // topics 必填属性
)

type XMLContent struct {
	XMLName xml.Name `xml:"xmap-content,omitempty"`
	Sheet   XMLSheet `xml:"sheet,omitempty"`
}

type XMLSheet struct {
	Topic *XMLTopic `xml:"topic,omitempty"`
}

type XMLTopic struct {
	Title    string            `xml:"title,omitempty"`
	Children *XMLTopicChildren `xml:"children,omitempty"`
}

type XMLTopicChildren struct {
	// TypedTopics 存在是为了设置 topics 这一层的 type 属性，必填，否则生成的脑图不展示子节点
	// 从数据结构上来说可以直接用 []XMLTopic `xml:"topics>topic"` 替代，但这样无法在 topics 这一层设置 type 属性
	// 另外 topics 和 topic 都有单独属性可填，所以还是需要单独设置结构
	TypedTopics *XMLTypedTopics `xml:"topics,omitempty"`
}

type XMLTypedTopics struct {
	Type   XMLTopicType `xml:"type,attr"`
	Topics []*XMLTopic  `xml:"topic"` // 这里为 topic，实际上为数组
}

// addChildTopic 返回 new topic 指针
// 不管 children 下 title 是否已存在，均新增
func (t *XMLTopic) addChildTopic(title string, topicType XMLTopicType, ignoreIfExist bool) *XMLTopic {
	if t.Children == nil {
		t.Children = &XMLTopicChildren{}
	}
	if t.Children.TypedTopics == nil {
		t.Children.TypedTopics = &XMLTypedTopics{}
	}
	t.Children.TypedTopics.Type = topicType

	var newTopic *XMLTopic
	// already exist, return directly
	if ignoreIfExist {
		for _, topic := range t.Children.TypedTopics.Topics {
			if topic.Title == title {
				return topic
			}
		}
	}
	// not exist, create a new topic
	newTopic = &XMLTopic{Title: title}
	t.Children.TypedTopics.Topics = append(t.Children.TypedTopics.Topics, newTopic)
	return newTopic
}

// AddAttachedChildTopic
func (t *XMLTopic) AddAttachedChildTopic(title string, ignoreIfExist ...bool) *XMLTopic {
	ignore := false
	if len(ignoreIfExist) > 0 {
		ignore = true
	}
	return t.addChildTopic(title, TopicsTypeAttached, ignore)
}

func ParseXML(r io.Reader) (XMLContent, error) {
	var content XMLContent
	if err := xml.NewDecoder(r).Decode(&content); err != nil {
		return XMLContent{}, err
	}
	return content, nil
}

func Export(w io.Writer, content XMLContent, filename string) error {
	// set headers to http ResponseWriter `w` before write into `w`.
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Add("Content-Disposition", "attachment;fileName="+url.QueryEscape(filename+".xmind"))
		//rw.Header().Add("Content-Type", "application/vnd.ms-xmind")
	}

	// 创建临时目录用于制作 .xmind (zip)
	tmpDir := os.TempDir()

	// 创建 content.xml
	contentXmlFilePath := filepath.Join(tmpDir, "content.xml")
	contentXmlBytes, err := xml.Marshal(&content)
	if err != nil {
		return fmt.Errorf("failed to generate content.xml content, err: %v", err)
	}
	if err := filehelper.CreateFile(contentXmlFilePath, string(contentXmlBytes), 0644); err != nil {
		return fmt.Errorf("failed to create content.xml, err: %v", err)
	}

	// 创建 META-INF/manifest.xml
	manifestXmlFilePath := filepath.Join(tmpDir, "META-INF", "manifest.xml")
	if err := filehelper.CreateFile(manifestXmlFilePath, fixedManifestXmlFileContent, 0644); err != nil {
		return fmt.Errorf("failed to create META-INF/manifest.xml, err: %v", err)
	}

	// 创建 meta.xml
	metaXmlFilePath := filepath.Join(tmpDir, "meta.xml")
	if err := filehelper.CreateFile(metaXmlFilePath, fixedMetaXmlFileContent, 0644); err != nil {
		return fmt.Errorf("failed to create meta.xml, err: %v", err)
	}

	// 制作 .xmind (.zip 压缩文件)
	if err := archiver.Zip.Write(w, []string{contentXmlFilePath, filepath.Dir(manifestXmlFilePath), metaXmlFilePath}); err != nil {
		return fmt.Errorf("failed to create .xmind, err: %v", err)
	}

	return nil
}
