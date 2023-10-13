#![feature(once_cell)]

#[cfg(not(target_env = "msvc"))]
use tikv_jemallocator::Jemalloc;

#[cfg(not(target_env = "msvc"))]
#[global_allocator]
static ALLOCATOR: Jemalloc = Jemalloc;

mod batch;
mod chunk;
mod types;
mod utils;
