#[allow(dead_code)]
mod api;
mod app;
#[allow(dead_code)]
mod theme;
#[allow(dead_code)]
mod types;
mod views;

use std::io;

use app::App;

fn main() -> io::Result<()> {
    App::run()
}
