//! Generate raw descriptor inventories and typed protobuf mutation setters.

use std::{collections::BTreeMap, fmt::Write};

use prost::Message;
use umpire_runtime::prost_types::{
    DescriptorProto, FieldDescriptorProto, FileDescriptorSet,
    field_descriptor_proto::{Label, Type},
};

/// Request generation identities and explicitly admitted roots.
pub struct Config<'a> {
    /// Fully qualified protobuf service name.
    pub service: &'a str,
    /// Rust module containing generated packages; empty for the current module.
    pub rust_type_root: &'a str,
    /// Generated method enum name.
    pub method_enum: &'a str,
    /// Public request field inventory function name.
    pub fields_function: &'a str,
    /// Public request mutation function name.
    pub mutate_function: &'a str,
    /// Source method names whose request closures are admitted for mutation.
    pub request_roots: &'a [&'a str],
}
const MAX_DEPTH: usize = 4;

struct Node<'a> {
    descriptor: &'a DescriptorProto,
    rust: String,
}

#[derive(Clone)]
struct Step {
    parent: String,
    field: FieldDescriptorProto,
    element: bool,
}

struct Field {
    path: String,
    declaration: String,
    kind: String,
    optional: bool,
    refusal: Option<String>,
    steps: Vec<Step>,
}

struct Schema<'a> {
    nodes: BTreeMap<String, Node<'a>>,
    enums: BTreeMap<String, Vec<i32>>,
}

/// Generate descriptor field inventories and typed request mutation setters.
pub fn generate(bytes: &[u8], config: &Config<'_>) -> Result<String, String> {
    let descriptors = FileDescriptorSet::decode(bytes).map_err(|error| error.to_string())?;
    let schema = Schema::new(&descriptors, config.rust_type_root);
    let (package, name) =
        config.service.rsplit_once('.').ok_or("service must be fully qualified")?;
    let service = descriptors
        .file
        .iter()
        .filter(|file| file.package.as_deref() == Some(package))
        .flat_map(|file| &file.service)
        .find(|service| service.name.as_deref() == Some(name))
        .ok_or_else(|| format!("missing service descriptor: {}", config.service))?;
    let method_enum = config.method_enum;
    let mut roots = Vec::new();
    for method in config.request_roots {
        let proto = service
            .method
            .iter()
            .find(|rpc| rpc.name.as_deref() == Some(method))
            .and_then(|rpc| rpc.input_type.as_deref())
            .ok_or_else(|| format!("missing request descriptor: {}.{method}", config.service))?
            .trim_start_matches('.')
            .to_owned();
        let mut fields = Vec::new();
        schema.walk(&proto, &proto, &[], &[], &mut fields)?;
        roots.push((*method, proto, fields));
    }
    let mut out = template(
        config,
        "use ::umpire_catalog::{WireField,WireKind,WireMutation,WireValue,GenerationError};\n\
         /// Raw request fields, including descriptor fields ignored by semantic catalogs.\n\
         pub fn catalog_request_fields(method: CatalogServiceMethod) -> Vec<WireField> { match method {\n",
    );
    for (method, _, fields) in &roots {
        writeln!(out, "{method_enum}::{method} => vec![").map_err(|error| error.to_string())?;
        for field in fields {
            let refusal = field
                .refusal
                .as_ref()
                .map_or_else(|| "None".to_owned(), |reason| format!("Some({reason:?}.into())"));
            writeln!(out,"WireField{{path:{:?}.into(),declaration:{:?}.into(),kind:{},optional:{},refusal:{refusal}}},",field.path,field.declaration,field.kind,field.optional).map_err(|error| error.to_string())?;
        }
        out.push_str("],\n");
    }
    if roots.len() < service.method.len() {
        out.push_str("_ => vec![WireField{path:method.name().into(),declaration:method.name().into(),kind:WireKind::Message,optional:false,refusal:Some(\"request root is outside the live probe surface\".into())}],\n");
    }
    out.push_str("}}\n");
    out.push_str(&template(config,"/// Apply one typed descriptor mutation to a decoded request.\npub fn mutate_catalog_request(method:CatalogServiceMethod,bytes:&[u8],path:&str,mutation:&WireMutation)->Result<Vec<u8>,GenerationError>{match method{\n"));
    for (method, proto, _) in &roots {
        let rust = &schema.node(proto)?.rust;
        let factory = snake(method);
        writeln!(out,"{method_enum}::{method}=>{{let mut message=<{rust} as ::prost::Message>::decode(bytes).map_err(|_|GenerationError::InvalidEncoding)?;mutate_{factory}(&mut message,path,mutation)?;Ok(::prost::Message::encode_to_vec(&message))}},").map_err(|error| error.to_string())?;
    }
    if roots.len() < service.method.len() {
        out.push_str("_=>Err(GenerationError::UnsupportedType),\n");
    }
    out.push_str("}}\n");
    for (method, proto, fields) in &roots {
        let rust = &schema.node(proto)?.rust;
        let factory = snake(method);
        writeln!(out,"fn mutate_{factory}(message:&mut {rust},path:&str,mutation:&WireMutation)->Result<(),GenerationError>{{match path{{").map_err(|error| error.to_string())?;
        for field in fields.iter().filter(|field| field.refusal.is_none()) {
            write!(out, "{:?}=>{{let node0=message;", field.path)
                .map_err(|error| error.to_string())?;
            schema.setter(field, &mut out)?;
            out.push_str("Ok(())},\n");
        }
        out.push_str("_=>Err(GenerationError::UnsupportedType),\n}}\n");
    }
    if roots.iter().any(|(_, _, fields)| {
        fields
            .iter()
            .any(|field| field.kind == "WireKind::Map" || field.kind == "WireKind::Repeated")
    }) {
        out.push_str("fn wire_collection_len(mutation:&WireMutation)->Result<usize,GenerationError>{let WireMutation::LimitPlusOne{limit}=mutation else{return Err(GenerationError::InapplicableMutation)};let length=limit.checked_add(1).ok_or(GenerationError::SizeLimit)?;if length>4096{return Err(GenerationError::SizeLimit)}usize::try_from(length).map_err(|_|GenerationError::SizeLimit)}\n");
    }
    Ok(out)
}

impl<'a> Schema<'a> {
    fn new(descriptors: &'a FileDescriptorSet, rust_type_root: &str) -> Self {
        let mut schema = Self { nodes: BTreeMap::new(), enums: BTreeMap::new() };
        for file in &descriptors.file {
            let package = file.package.as_deref().unwrap_or_default();
            let rust = if package == "google.protobuf" {
                "::umpire_runtime::prost_types".into()
            } else if rust_type_root.is_empty() {
                package.replace('.', "::")
            } else {
                format!("{rust_type_root}::{}", package.replace('.', "::"))
            };
            schema.collect(package, &rust, &file.message_type);
            for enumeration in &file.enum_type {
                schema.enums.insert(
                    format!("{package}.{}", enumeration.name.as_deref().unwrap_or_default()),
                    enumeration.value.iter().filter_map(|value| value.number).collect(),
                );
            }
        }
        schema
    }

    fn collect(&mut self, proto: &str, rust: &str, messages: &'a [DescriptorProto]) {
        for descriptor in messages {
            let name = descriptor.name.as_deref().unwrap_or_default();
            let proto = format!("{proto}.{name}");
            self.nodes.insert(
                proto.clone(),
                Node {
                    descriptor,
                    rust: umpire_catalog_syntax::descriptor::well_known_primitive(&proto)
                        .map_or_else(|| format!("{rust}::{name}"), str::to_owned),
                },
            );
            for enumeration in &descriptor.enum_type {
                self.enums.insert(
                    format!("{proto}.{}", enumeration.name.as_deref().unwrap_or_default()),
                    enumeration.value.iter().filter_map(|value| value.number).collect(),
                );
            }
            self.collect(&proto, &format!("{rust}::{}", snake(name)), &descriptor.nested_type);
        }
    }

    fn node(&self, name: &str) -> Result<&Node<'a>, String> {
        self.nodes
            .get(name.trim_start_matches('.'))
            .ok_or_else(|| format!("missing descriptor message: {name}"))
    }

    fn walk(
        &self,
        proto: &str,
        path: &str,
        steps: &[Step],
        parents: &[String],
        fields: &mut Vec<Field>,
    ) -> Result<(), String> {
        let node = self.node(proto)?;
        let refusal = if parents.iter().any(|parent| parent == proto) {
            Some("recursive descriptor edge; deeper fields were not expanded")
        } else if parents.len() >= MAX_DEPTH {
            Some("descriptor depth limit reached; deeper fields were not expanded")
        } else {
            None
        };
        let mut parents = parents.to_vec();
        parents.push(proto.into());
        for field in &node.descriptor.field {
            let name = field.name.as_deref().ok_or("unnamed descriptor field")?;
            let path = format!("{path}.{name}");
            let mut steps = steps.to_vec();
            steps.push(Step { parent: proto.into(), field: field.clone(), element: false });
            fields.push(Field {
                path: path.clone(),
                declaration: format!("{proto}.{name}"),
                kind: self.kind(field)?,
                optional: self.optional(proto, field)?,
                refusal: refusal.map(str::to_owned),
                steps: steps.clone(),
            });
            if refusal.is_some() {
                continue;
            }
            if field.r#type == Some(Type::Message as i32) {
                let child = field
                    .type_name
                    .as_deref()
                    .ok_or("message field without target")?
                    .trim_start_matches('.');
                self.walk(child, &path, &steps, &parents, fields)?;
            } else if field.label == Some(Label::Repeated as i32) {
                let mut scalar = field.clone();
                scalar.label = Some(Label::Optional as i32);
                steps.push(Step { parent: proto.into(), field: scalar.clone(), element: true });
                fields.push(Field {
                    path: format!("{path}.value"),
                    declaration: format!("{proto}.{name}"),
                    kind: self.kind(&scalar)?,
                    optional: false,
                    refusal: None,
                    steps,
                });
            }
        }
        Ok(())
    }

    fn optional(&self, parent: &str, field: &FieldDescriptorProto) -> Result<bool, String> {
        Ok(!self.is_map_entry(parent)?
            && field.label != Some(Label::Repeated as i32)
            && (field.r#type == Some(Type::Message as i32)
                || field.oneof_index.is_some()
                || field.proto3_optional == Some(true)))
    }

    fn is_map_entry(&self, name: &str) -> Result<bool, String> {
        Ok(self
            .node(name)?
            .descriptor
            .options
            .as_ref()
            .is_some_and(|options| options.map_entry == Some(true)))
    }

    fn map(&self, field: &FieldDescriptorProto) -> Result<bool, String> {
        if field.r#type == Some(Type::Message as i32) {
            self.is_map_entry(field.type_name.as_deref().ok_or("missing message target")?)
        } else {
            Ok(false)
        }
    }

    fn kind(&self, field: &FieldDescriptorProto) -> Result<String, String> {
        if self.map(field)? {
            return Ok("WireKind::Map".into());
        }
        if field.label == Some(Label::Repeated as i32) {
            return Ok("WireKind::Repeated".into());
        }
        let kind = match Type::try_from(field.r#type.unwrap_or_default())
            .map_err(|_| "invalid field type")?
        {
            Type::String => "String",
            Type::Bytes => "Bytes",
            Type::Bool => "Bool",
            Type::Double | Type::Float => "Float",
            Type::Uint32 | Type::Uint64 | Type::Fixed32 | Type::Fixed64 => "Unsigned",
            Type::Int32
            | Type::Int64
            | Type::Sint32
            | Type::Sint64
            | Type::Sfixed32
            | Type::Sfixed64 => "Signed",
            Type::Message => "Message",
            Type::Enum => {
                let name = field
                    .type_name
                    .as_deref()
                    .ok_or("missing enum target")?
                    .trim_start_matches('.');
                let values = self
                    .enums
                    .get(name)
                    .ok_or_else(|| format!("missing enum descriptor: {name}"))?;
                return Ok(format!("WireKind::Enum{{defined:vec!{values:?}}}"));
            }
            Type::Group => return Err("protobuf groups are unsupported".into()),
        };
        Ok(format!("WireKind::{kind}"))
    }

    fn member(&self, step: &Step) -> Result<String, String> {
        if self.is_map_entry(&step.parent)? {
            return Ok(if step.field.number == Some(1) { "0" } else { "1" }.into());
        }
        let name = snake(step.field.name.as_deref().ok_or("unnamed field")?);
        Ok(format!("r#{name}"))
    }

    fn oneof(&self, step: &Step) -> Result<Option<(String, String, String)>, String> {
        if step.field.proto3_optional == Some(true) {
            return Ok(None);
        }
        let Some(index) = step.field.oneof_index else {
            return Ok(None);
        };
        let node = self.node(&step.parent)?;
        let name = node
            .descriptor
            .oneof_decl
            .get(usize::try_from(index).map_err(|_| "invalid oneof index")?)
            .and_then(|oneof| oneof.name.as_deref())
            .ok_or("missing oneof name")?;
        let (module, owner) = node.rust.rsplit_once("::").ok_or("invalid message Rust path")?;
        Ok(Some((
            format!("r#{}", snake(name)),
            format!("{module}::{}::{}", snake(owner), camel(name)),
            camel(step.field.name.as_deref().ok_or("unnamed oneof field")?),
        )))
    }

    fn setter(&self, field: &Field, out: &mut String) -> Result<(), String> {
        let mut cleanups = Vec::new();
        for (index, step) in field.steps.iter().enumerate() {
            let parent = format!("node{index}");
            let child = format!("node{}", index + 1);
            let leaf = index + 1 == field.steps.len();
            if leaf {
                self.leaf(step, &parent, &field.kind, out)?;
                break;
            }
            if let Some((member, ty, variant)) = self.oneof(step)? {
                absent_guard(
                    &format!("!matches!({parent}.{member},Some({ty}::{variant}(_)))"),
                    &cleanups,
                    out,
                )?;
                write!(out,"if !matches!({parent}.{member},Some({ty}::{variant}(_))){{{parent}.{member}=Some({ty}::{variant}(Default::default()));}}let Some({ty}::{variant}({child}))={parent}.{member}.as_mut() else{{return Err(GenerationError::UnsupportedType)}};").map_err(|error| error.to_string())?;
            } else {
                let member = self.member(step)?;
                if self.map(&step.field)? {
                    absent_guard(&format!("{parent}.{member}.is_empty()"), &cleanups, out)?;
                    write!(out,"let key{index}={parent}.{member}.keys().next().cloned().unwrap_or_default();let mut entry{index}=(key{index}.clone(),{parent}.{member}.remove(&key{index}).unwrap_or_default());let {child}=&mut entry{index};").map_err(|error| error.to_string())?;
                    cleanups
                        .push(format!("{parent}.{member}.insert(entry{index}.0,entry{index}.1);"));
                } else if step.field.label == Some(Label::Repeated as i32) {
                    absent_guard(&format!("{parent}.{member}.is_empty()"), &cleanups, out)?;
                    write!(out,"if {parent}.{member}.is_empty(){{{parent}.{member}.push(Default::default());}}let {child}={parent}.{member}.first_mut().ok_or(GenerationError::UnsupportedType)?;").map_err(|error| error.to_string())?;
                } else if self.optional(&step.parent, &step.field)? {
                    absent_guard(&format!("{parent}.{member}.is_none()"), &cleanups, out)?;
                    write!(
                        out,
                        "let {child}={parent}.{member}.get_or_insert_with(Default::default);"
                    )
                    .map_err(|error| error.to_string())?;
                } else {
                    write!(out, "let {child}=&mut {parent}.{member};")
                        .map_err(|error| error.to_string())?;
                }
            }
        }
        for cleanup in cleanups.iter().rev() {
            out.push_str(cleanup);
        }
        Ok(())
    }

    fn leaf(&self, step: &Step, parent: &str, kind: &str, out: &mut String) -> Result<(), String> {
        if step.element {
            write!(out, "WireValue::mutate({parent},&{kind},mutation)?;")
                .map_err(|error| error.to_string())?;
            return Ok(());
        }
        if let Some((member, ty, variant)) = self.oneof(step)? {
            write!(
                out,
                "if matches!(mutation,WireMutation::Absent){{if matches!({parent}.{member},Some({ty}::{variant}(_))){{{parent}.{member}=None;}}}}else{{"
            )
            .map_err(|error| error.to_string())?;
            if step.field.r#type == Some(Type::Message as i32) {
                write!(out,"if !matches!(mutation,WireMutation::Empty){{return Err(GenerationError::InapplicableMutation)}}{parent}.{member}=Some({ty}::{variant}(Default::default()));").map_err(|error| error.to_string())?;
            } else {
                write!(out,"if !matches!({parent}.{member},Some({ty}::{variant}(_))){{{parent}.{member}=Some({ty}::{variant}(Default::default()));}}let Some({ty}::{variant}(value))={parent}.{member}.as_mut() else{{return Err(GenerationError::UnsupportedType)}};WireValue::mutate(value,&{kind},mutation)?;").map_err(|error| error.to_string())?;
            }
            out.push('}');
            return Ok(());
        }
        let member = self.member(step)?;
        let target =
            if umpire_catalog_syntax::descriptor::well_known_primitive(&step.parent).is_some() {
                format!("(*{parent})")
            } else {
                format!("{parent}.{member}")
            };
        if self.map(&step.field)? {
            self.map_leaf(&step.field, &target, out)?;
        } else if step.field.label == Some(Label::Repeated as i32) {
            write!(out,"match mutation{{WireMutation::Absent|WireMutation::Empty=>{target}.clear(),WireMutation::LimitPlusOne{{..}}=>{{let length=wire_collection_len(mutation)?;{target}.clear();{target}.resize_with(length,Default::default);}},_=>return Err(GenerationError::InapplicableMutation)}}").map_err(|error| error.to_string())?;
        } else if step.field.r#type == Some(Type::Message as i32) {
            if self.optional(&step.parent, &step.field)? {
                write!(out,"match mutation{{WireMutation::Absent=>{target}=None,WireMutation::Empty=>{target}=Some(Default::default()),_=>return Err(GenerationError::InapplicableMutation)}}").map_err(|error| error.to_string())?;
            } else {
                write!(out,"match mutation{{WireMutation::Absent|WireMutation::Empty=>{target}=Default::default(),_=>return Err(GenerationError::InapplicableMutation)}}").map_err(|error| error.to_string())?;
            }
        } else if self.optional(&step.parent, &step.field)? {
            write!(out,"if matches!(mutation,WireMutation::Absent){{{target}=None;}}else{{WireValue::mutate({target}.get_or_insert_with(Default::default),&{kind},mutation)?;}}").map_err(|error| error.to_string())?;
        } else {
            write!(out, "WireValue::mutate(&mut {target},&{kind},mutation)?;")
                .map_err(|error| error.to_string())?;
        }
        Ok(())
    }

    fn map_leaf(
        &self,
        field: &FieldDescriptorProto,
        target: &str,
        out: &mut String,
    ) -> Result<(), String> {
        let entry = self.node(field.type_name.as_deref().ok_or("missing map entry")?)?;
        let key = entry
            .descriptor
            .field
            .iter()
            .find(|field| field.number == Some(1))
            .ok_or("missing map key")?;
        let key =
            match Type::try_from(key.r#type.unwrap_or_default()).map_err(|_| "invalid map key")? {
                Type::String => "index.to_string()",
                Type::Bool => "index!=0",
                _ => "index.try_into().map_err(|_|GenerationError::SizeLimit)?",
            };
        write!(out,"match mutation{{WireMutation::Absent|WireMutation::Empty=>{target}.clear(),WireMutation::LimitPlusOne{{..}}=>{{let length=wire_collection_len(mutation)?;").map_err(|error| error.to_string())?;
        if key == "index!=0" {
            out.push_str("if length>2{return Err(GenerationError::SizeLimit)}");
        }
        write!(out,"{target}.clear();for index in 0..length{{{target}.insert({key},Default::default());}}}},_=>return Err(GenerationError::InapplicableMutation)}}").map_err(|error| error.to_string())?;
        Ok(())
    }
}

fn absent_guard(condition: &str, cleanups: &[String], out: &mut String) -> Result<(), String> {
    write!(out, "if matches!(mutation,WireMutation::Absent)&&({condition}){{")
        .map_err(|error| error.to_string())?;
    for cleanup in cleanups.iter().rev() {
        out.push_str(cleanup);
    }
    out.push_str("return Ok(());}");
    Ok(())
}

fn snake(name: &str) -> String {
    let chars: Vec<_> = name.chars().collect();
    let mut result = String::new();
    for (index, ch) in chars.iter().enumerate() {
        if ch.is_uppercase()
            && index > 0
            && (chars[index - 1].is_lowercase()
                || chars[index - 1].is_ascii_digit()
                || chars.get(index + 1).is_some_and(|next| next.is_lowercase()))
        {
            result.push('_');
        }
        result.extend(ch.to_lowercase());
    }
    result
}

fn camel(name: &str) -> String {
    snake(name)
        .split('_')
        .filter(|part| !part.is_empty())
        .map(|part| {
            let mut chars = part.chars();
            chars
                .next()
                .map_or_else(String::new, |first| first.to_uppercase().chain(chars).collect())
        })
        .collect()
}

fn template(config: &Config<'_>, source: &str) -> String {
    source
        .replace("CatalogServiceMethod", config.method_enum)
        .replace("catalog_request_fields", config.fields_function)
        .replace("mutate_catalog_request", config.mutate_function)
}
