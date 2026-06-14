from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ObjectType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OBJECT_TYPE_UNSPECIFIC: _ClassVar[ObjectType]
    OBJECT_TYPE_VIEW: _ClassVar[ObjectType]
    OBJECT_TYPE_PROCEDURE: _ClassVar[ObjectType]
OBJECT_TYPE_UNSPECIFIC: ObjectType
OBJECT_TYPE_VIEW: ObjectType
OBJECT_TYPE_PROCEDURE: ObjectType

class ObjectInfo(_message.Message):
    __slots__ = ("name", "sql", "object_type")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SQL_FIELD_NUMBER: _ClassVar[int]
    OBJECT_TYPE_FIELD_NUMBER: _ClassVar[int]
    name: str
    sql: str
    object_type: ObjectType
    def __init__(self, name: _Optional[str] = ..., sql: _Optional[str] = ..., object_type: _Optional[_Union[ObjectType, str]] = ...) -> None: ...

class Lineage(_message.Message):
    __slots__ = ("target", "source")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    target: str
    source: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., source: _Optional[_Iterable[str]] = ...) -> None: ...

class ObjectLineage(_message.Message):
    __slots__ = ("name", "lineages")
    NAME_FIELD_NUMBER: _ClassVar[int]
    LINEAGES_FIELD_NUMBER: _ClassVar[int]
    name: str
    lineages: _containers.RepeatedCompositeFieldContainer[Lineage]
    def __init__(self, name: _Optional[str] = ..., lineages: _Optional[_Iterable[_Union[Lineage, _Mapping]]] = ...) -> None: ...

class LineageRequest(_message.Message):
    __slots__ = ("objects",)
    OBJECTS_FIELD_NUMBER: _ClassVar[int]
    objects: _containers.RepeatedCompositeFieldContainer[ObjectInfo]
    def __init__(self, objects: _Optional[_Iterable[_Union[ObjectInfo, _Mapping]]] = ...) -> None: ...

class LineageResponse(_message.Message):
    __slots__ = ("lineages",)
    LINEAGES_FIELD_NUMBER: _ClassVar[int]
    lineages: _containers.RepeatedCompositeFieldContainer[ObjectLineage]
    def __init__(self, lineages: _Optional[_Iterable[_Union[ObjectLineage, _Mapping]]] = ...) -> None: ...
