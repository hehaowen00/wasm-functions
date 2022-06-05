use rusqlite::{params, Connection, Result};
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

fn main() -> Result<()> {
    let req = std::env::var("REQUEST").unwrap();
    let req: Todo = serde_json::from_str(&req).unwrap();

    let db = Connection::open("./todos/todos.db")?;

    db.execute(
        "UPDATE todos SET desc = ?1, status = ?2 WHERE ID = ?3",
        params![req.desc, req.status, req.id],
    )?;

    let msg = "Success";

    println!("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: {}\r\n", msg.len());
    println!("{}", msg);

    Ok(())
}
