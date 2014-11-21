// Checks that anonymous structs are handled properly.
// TODO(zarko): check for the type edge (depends on naming CL)
//- @struct defines AnonStruct
//- AnonStruct.complete definition
//- AnonStruct.subkind struct
//- AnonStruct.node/kind record
struct { } S;
