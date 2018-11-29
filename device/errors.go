package device

// InternalError represents an error when something goes wrong, and its our fault.
const InternalError = 2

// DatabaseError is when some operation related to Database, such as insert or find,
// goes wrong and the task cannot proceed.
const DatabaseError = 3

// UserError occurs when there's an error because of user's action.
// An example would be providing devicealid input.
const UserError = 4
