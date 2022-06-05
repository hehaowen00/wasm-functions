use rusqlite::{Connection, OpenFlags, Result};
use serde::{Deserialize, Serialize};
use sqlite_vfs::{register, OpenAccess, OpenOptions, Vfs};
use std::fs;
use std::path::Path;

struct FsVfs;

impl Vfs for FsVfs {
    type File = fs::File;

    fn open(&self, path: &Path, opts: OpenOptions) -> Result<Self::File, std::io::Error> {
        let mut o = fs::OpenOptions::new();
        o.read(true).write(opts.access != OpenAccess::Read);

        match opts.access {
            OpenAccess::Create => {
                o.create(true);
            }
            OpenAccess::CreateNew => {
                o.create_new(true);
            }
            _ => {}
        }

        let f = o.open(path)?;

        Ok(f)
    }

    fn delete(&self, path: &Path) -> Result<(), std::io::Error> {
        std::fs::remove_file(path)
    }

    fn exists(&self, path: &Path) -> Result<bool, std::io::Error> {
        Ok(path.is_file())
    }
}

#[no_mangle]
pub extern "C" fn sqlite3_os_init() -> i32 {
    register("default", FsVfs {}).unwrap();
    return 0;
}

#[derive(Serialize, Deserialize)]
struct Todo {
    id: u64,
    desc: String,
    status: bool,
}

fn main() {
    let db = Connection::open("./todos/todos.db").unwrap();

    db.execute(
        "CREATE TABLE IF NOT EXISTS todos (
                id INTEGER PRIMARY KEY,
                desc TEXT NOT NULL,
                status INTEGER NOT NULL
            );",
        [],
    ).unwrap();

    let mut stmt = db.prepare("SELECT id, desc, status FROM todos").unwrap();

    let iter = stmt
        .query_map([], |row| {
            let v: i8 = row.get(2).unwrap();
            let status = if v == 0 { false } else { true };
            Ok(Todo {
                id: row.get(0).unwrap(),
                desc: row.get(1).unwrap(),
                status,
            })
        })
        .unwrap();

    let mut todos: Vec<Todo> = Vec::new();

    for item in iter {
        todos.push(item.unwrap());
    }

    let json = serde_json::to_string(&todos).unwrap();

    println!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nConnection: Keep-Alive\r\nContent-Length: {}\r\n", json.len());
    println!("{}", json);
}
