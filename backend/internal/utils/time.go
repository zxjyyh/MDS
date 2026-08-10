package utils

import "time"

// 项目内所有写入数据库的时间戳都必须经由这里，保证格式统一。
//
// 背景：SQLite 的 `DEFAULT (datetime('now'))` 返回的是 **UTC** 且不带时区偏移
// （形如 `2026-08-10 04:00:00`），而 Go 侧写的是带偏移的本地时间
// （形如 `2026-08-10T12:00:00+08:00`）。两者混在同一列里，排序和比较都会出错。
//
// 两种格式恰好可以区分（有无 'T'、有无偏移），
// 因此 database.NormalizeLegacyTimestamps 能在启动时把历史 UTC 值就地修正。

// NowString 返回当前本地时间的 RFC3339 表示，用于写入数据库。
func NowString() string {
	return time.Now().Format(time.RFC3339)
}

// TodayString 返回当前本地日期（YYYY-MM-DD），用于纯日期字段。
func TodayString() string {
	return time.Now().Format("2006-01-02")
}

// SQLiteDefaultLayout 是 SQLite `datetime('now')` 产生的格式（UTC，无偏移）。
const SQLiteDefaultLayout = "2006-01-02 15:04:05"
