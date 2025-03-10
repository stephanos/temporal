// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package propmodel

import (
	"cmp"
	"fmt"

	"go.temporal.io/server/common/persistence"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/tasktoken"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Namespace struct {
		Model[Namespace]
		Cluster Scope[Cluster]

		tokenSerializer *tasktoken.Serializer
	}

	NamespaceID string
)

func ImportNamespace(
	env *Env,
	request *persistence.CreateNamespaceRequest,
) error {
	evt := NewEvent(request)
	evt.StringFunc = func() string {
		return fmt.Sprintf("CreateNamespaceRequest{Namespace: %v}", request.Namespace.Info.Name)
	}
	return env.Send(evt)
}

func (n *Namespace) OnRequest(
	req requestMsg,
) (
	nsName ID,
	nsID NamespaceID,
) {
	return n.extractIDs(req)
}

func (n *Namespace) OnResponse(
	req requestMsg,
	resp responseMsg,
) (
	nsName ID,
	nsID NamespaceID,
) {
	nsName, nsID = n.extractIDs(req)
	if nsName == "" || nsID == "" {
		respNsName, respNsID := n.extractIDs(resp)
		nsName = cmp.Or(nsName, respNsName)
		nsID = cmp.Or(nsID, respNsID)
	}
	return
}

func (n *Namespace) extractIDs(
	msg proto.Message,
) (
	nsName ID,
	nsID NamespaceID,
) {
	// (1) try finding the namespace name
	nsName = ID(findProtoValueByNameType[string](msg, "namespace", protoreflect.StringKind))
	if nsName == "temporal-system" {
		// ignore the system namespace
		nsName = Ignored
	}

	// (2) try finding the namespace id
	// TODO: wrangle the existing routing code to be reusable here
	nsID = NamespaceID(findProtoValueByNameType[string](msg, "namespace_id", protoreflect.StringKind))
	if nsID == "" {
		if n.tokenSerializer == nil {
			n.tokenSerializer = tasktoken.NewSerializer()
		}
		taskToken := findProtoValueByNameType[[]byte](msg, "task_token", protoreflect.BytesKind)
		if taskToken != nil {
			t, _ := n.tokenSerializer.Deserialize(taskToken)
			nsID = NamespaceID(t.GetNamespaceId())
		}
	}

	return
}

func (n *Namespace) OnCreateRequest(
	req *persistence.CreateNamespaceRequest,
) (
	nsName ID,
	nsID NamespaceID,
) {
	nsName = ID(req.Namespace.Info.Name)
	nsID = NamespaceID(req.Namespace.Info.Id)
	return
}
