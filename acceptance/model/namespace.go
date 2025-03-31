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

package model

import (
	"cmp"

	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/tasktoken"
	"go.temporal.io/server/common/testing/stamp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	Namespace struct {
		stamp.Model[*Namespace]
		stamp.Scope[*Cluster]

		InternalID      string
		tokenSerializer *tasktoken.Serializer
	}
)

// ==== Identity Handlers

func (n *Namespace) IdRequest(req RequestMsg, _ Incoming) stamp.ID {
	nsName, nsID := n.extractIDs(req)
	n.InternalID = nsID
	return nsName
}

func (n *Namespace) IdResponse(req RequestMsg, resp ResponseMsg) stamp.ID {
	nsName, nsID := n.extractIDs(req)
	if nsName == "" || nsID == "" {
		respNsName, respNsID := n.extractIDs(resp)
		nsName = cmp.Or(nsName, respNsName)
		nsID = cmp.Or(nsID, respNsID)
	}
	return nsName
}

func (n *Namespace) IdCreateRequest(req *persistence.CreateNamespaceRequest) stamp.ID {
	n.InternalID = req.Namespace.Info.Id
	return stamp.ID(req.Namespace.Info.Name)
}

func (n *Namespace) IdInternalCreateRequest(req *persistence.InternalCreateNamespaceRequest) stamp.ID {
	// InternalCreateNamespaceRequest is a special case for the test setup
	// which creates the namespace directly; instead of through the API.
	n.InternalID = req.ID
	return stamp.ID(req.Name)
}

func (n *Namespace) extractIDs(msg proto.Message) (nsName stamp.ID, nsID string) {
	// (1) try finding the namespace name
	nsName = stamp.ID(findProtoValueByNameType[string](msg, "namespace", protoreflect.StringKind))

	// (2) try finding the namespace id
	// TODO: wrangle the existing routing code to be reusable here
	nsID = findProtoValueByNameType[string](msg, "namespace_id", protoreflect.StringKind)
	if nsID == "" {
		if n.tokenSerializer == nil {
			n.tokenSerializer = tasktoken.NewSerializer()
		}
		taskToken := findProtoValueByNameType[[]byte](msg, "task_token", protoreflect.BytesKind)
		if taskToken != nil {
			t, _ := n.tokenSerializer.Deserialize(taskToken)
			nsID = t.GetNamespaceId()
		}
	}

	return
}

// ==== Action handlers
