package multiplexer

import "gitlab.com/mainops/uniShell/internal/session"

type Metadata = session.Metadata

var MetadataPath = session.MetadataPath
var WriteMetadata = session.WriteMetadata
var ReadMetadata = session.ReadMetadata
var RemoveMetadata = session.RemoveMetadata
