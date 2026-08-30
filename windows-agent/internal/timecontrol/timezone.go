package timecontrol

// Bundle the IANA timezone database. Windows does not provide IANA names such
// as Asia/Ho_Chi_Minh to Go's time.LoadLocation by default.
import _ "time/tzdata"
