// source: fixture/public/model.proto

package godescriptors

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func init() {
	dependency, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("fixture/dependency.proto"),
		Package: proto.String("fixture.shared"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Dependency"),
		}},
	}, protoregistry.GlobalFiles)
	if err != nil {
		panic(err)
	}
	if err := protoregistry.GlobalFiles.RegisterFile(dependency); err != nil {
		panic(err)
	}
	root, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       proto.String("fixture/public/model.proto"),
		Package:    proto.String("fixture.public"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"fixture/dependency.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Model"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("dependency"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".fixture.shared.Dependency"),
			}},
		}},
	}, protoregistry.GlobalFiles)
	if err != nil {
		panic(err)
	}
	if err := protoregistry.GlobalFiles.RegisterFile(root); err != nil {
		panic(err)
	}
}
