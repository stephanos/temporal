// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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
	"go.temporal.io/server/common/persistence"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/tasktoken"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	tokenSerializer = tasktoken.NewSerializer()
)

type (
	Namespace struct {
		Model[Namespace]
		Cluster Scope[Cluster]

		internalID string
	}

	NamespaceID string
)

func ImportNamespace(
	env *Env,
	request *persistence.CreateNamespaceRequest,
) {
	env.Send(request)
}

// TODO: some responses don't contain the necessary info anymore; still need the request here, too
func (n *Namespace) IdGRPC(
	// TODO: allow using any variable as a parameter?
	msg proto.Message,
) ID {
	// try finding the namespace name
	nsName := findProtoValueByNameType[string](msg, "namespace", protoreflect.StringKind)
	if nsName == "temporal-system" {
		return Ignored
	}
	if nsName != "" {
		return ID(nsName)
	}

	// try finding the namespace id (only works for existing models)
	if currentID := n.GetID(); currentID != "" {
		// TODO: wrangle the existing routing code to be reusable here
		nsID := findProtoValueByNameType[string](msg, "namespace_id", protoreflect.StringKind)
		if nsID == "" {
			taskToken := findProtoValueByNameType[[]byte](msg, "task_token", protoreflect.BytesKind)
			if taskToken != nil {
				t, _ := tokenSerializer.Deserialize(taskToken)
				nsID = t.GetNamespaceId()
			}
		}
		if n.internalID == nsID {
			return ID(currentID)
		}
	}

	return Unknown
}

func (n *Namespace) IdPersistence(
	req *persistence.CreateNamespaceRequest,
) ID {
	return ID(req.Namespace.Info.Name)
}

func (n *Namespace) OnCreateRequest(
	req *persistence.CreateNamespaceRequest,
) (
	nsID NamespaceID,
) {
	n.internalID = req.Namespace.Info.Id
	nsID = NamespaceID(n.internalID)
	return
}
