use std::{
    fs::File,
    io::Read,
    path::{Path, PathBuf},
    process::Command,
    thread::sleep,
    time::Duration,
};

use anyhow::Result;
use assert_cmd::prelude::*;
use bpfd_api::util::directories::BYTECODE_IMAGE_CONTENT_STORE;
use log::debug;
use predicates::str::is_empty;

const NS_NAME: &str = "bpfd-int-test";

const HOST_VETH: &str = "veth-bpfd-host";
const NS_VETH: &str = "veth-bpfd-ns";

// The default prefix can be overriden by setting the BPFD_IP_PREFIX environment variable
const DEFAULT_IP_PREFIX: &str = "172.37.37";
const IP_MASK: &str = "24";
const HOST_IP_ID: &str = "1";
const NS_IP_ID: &str = "2";

pub const DEFAULT_BPFD_IFACE: &str = HOST_VETH;

const PING_FILE_NAME: &str = "/tmp/bpfd_ping.log";
const TRACE_PIPE_FILE_NAME: &str = "/tmp/bpfd_trace_pipe.log";

#[derive(Debug)]
pub enum LoadType {
    Image,
    File,
}

pub static LOAD_TYPES: &[LoadType] = &[LoadType::Image, LoadType::File];

const XDP_PASS_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/xdp_pass:latest";
const TC_PASS_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/tc_pass:latest";
const TRACEPOINT_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/tracepoint:latest";
const UPROBE_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/uprobe:latest";
const URETPROBE_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/uretprobe:latest";
const KPROBE_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/kprobe:latest";
const KRETPROBE_IMAGE_LOC: &str = "quay.io/bpfd-bytecode/kretprobe:latest";

const XDP_PASS_FILE_LOC: &str = "tests/integration-test/bpf/.output/xdp_pass.bpf.o";
const TC_PASS_FILE_LOC: &str = "tests/integration-test/bpf/.output/tc_pass.bpf.o";
const TRACEPOINT_FILE_LOC: &str = "tests/integration-test/bpf/.output/tp_openat.bpf.o";
const UPROBE_FILE_LOC: &str = "tests/integration-test/bpf/.output/uprobe.bpf.o";
const URETPROBE_FILE_LOC: &str = "tests/integration-test/bpf/.output/uprobe.bpf.o";
const KPROBE_FILE_LOC: &str = "tests/integration-test/bpf/.output/kprobe.bpf.o";
const KRETPROBE_FILE_LOC: &str = "tests/integration-test/bpf/.output/kprobe.bpf.o";
/// Exit on panic as well as the passing of a test
#[derive(Debug)]
pub struct ChildGuard {
    name: &'static str,
    child: std::process::Child,
}

impl Drop for ChildGuard {
    fn drop(&mut self) {
        debug!("stopping {}", self.name);
        if let Err(e) = self.child.kill() {
            println!("Could not kill {}: {e}", self.name);
        }
        if let Err(e) = self.child.wait() {
            println!("Could not wait for {}: {e}", self.name);
        }
    }
}

/// Spawn a bpfd process
pub fn start_bpfd() -> Result<ChildGuard> {
    debug!("Starting bpfd");
    let bpfd_process = Command::cargo_bin("bpfd")?.spawn().map(|c| ChildGuard {
        name: "bpfd",
        child: c,
    })?;

    // Wait for up to 5 seconds for bpfd to be ready
    sleep(Duration::from_millis(100));
    for i in 1..51 {
        if let Err(e) = Command::cargo_bin("bpfctl")?.args(["list"]).ok() {
            if i == 50 {
                panic!("bpfd not ready after {} ms. Error:\n{}", i * 100, e);
            } else {
                sleep(Duration::from_millis(100));
            }
        } else {
            break;
        }
    }
    debug!("Successfully Started bpfd");

    Ok(bpfd_process)
}

/// Install an xdp_pass program with bpfctl
pub fn add_xdp_pass(
    iface: &str,
    priority: u32,
    globals: Option<Vec<&str>>,
    proceed_on: Option<Vec<&str>>,
    load_type: &LoadType,
) -> Result<String> {
    let p = priority.to_string();

    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => {
            args.extend(["--image-url", XDP_PASS_IMAGE_LOC, "--pull-policy", "Always"])
        }
        LoadType::File => args.extend(["-s", "pass", "--path", XDP_PASS_FILE_LOC]),
    }

    args.extend(["xdp", "--iface", iface, "--priority", p.as_str()]);

    if let Some(p_o) = proceed_on {
        args.push("--proceed-on");
        args.extend(p_o);
    }

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added xdp_pass program: {:?} from: {:?}",
        uuid, load_type
    );

    Ok(uuid.to_string())
}

/// Install a tc_pass program with bpfctl
pub fn add_tc_pass(
    direction: &str,
    iface: &str,
    priority: u32,
    globals: Option<Vec<&str>>,
    proceed_on: Option<Vec<&str>>,
    load_type: &LoadType,
) -> Result<String> {
    let p = priority.to_string();

    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => {
            args.extend(["--image-url", TC_PASS_IMAGE_LOC, "--pull-policy", "Always"])
        }
        LoadType::File => args.extend(["-s", "pass", "--path", TC_PASS_FILE_LOC]),
    }

    args.extend([
        "tc",
        "--direction",
        direction,
        "--iface",
        iface,
        "--priority",
        p.as_str(),
    ]);

    if let Some(p_o) = proceed_on {
        args.push("--proceed-on");
        args.extend(p_o);
    }

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added tc {} program: {:?} from: {:?}",
        direction, uuid, load_type
    );
    Ok(uuid.to_string())
}

/// Install a tracepoint program with bpfctl
pub fn add_tracepoint(globals: Option<Vec<&str>>, load_type: &LoadType) -> Result<String> {
    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => args.extend([
            "--image-url",
            TRACEPOINT_IMAGE_LOC,
            "--pull-policy",
            "Always",
        ]),
        LoadType::File => args.extend(["-s", "enter_openat", "--path", TRACEPOINT_FILE_LOC]),
    }

    args.extend(["tracepoint", "--tracepoint", "syscalls/sys_enter_openat"]);

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added tracepoint program: {:?} from: {:?}",
        uuid, load_type
    );
    Ok(uuid.to_string())
}

/// Attach a uprobe program to bpfctl with bpfctl
pub fn add_uprobe(globals: Option<Vec<&str>>, load_type: &LoadType) -> Result<String> {
    let bpfctl_cmd = Command::cargo_bin("bpfctl")?;
    let bpfctl_path = bpfctl_cmd.get_program().to_str().unwrap();

    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => {
            args.extend(["--image-url", UPROBE_IMAGE_LOC, "--pull-policy", "Always"])
        }
        LoadType::File => args.extend(["-s", "my_uprobe", "--path", UPROBE_FILE_LOC]),
    }

    args.extend(["uprobe", "-f", "main", "-t", bpfctl_path]);

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added uprobe program: {:?} from: {:?}",
        uuid, load_type
    );
    Ok(uuid.to_string())
}

/// Attach a uretprobe program to bpfctl with bpfctl
pub fn add_uretprobe(globals: Option<Vec<&str>>, load_type: &LoadType) -> Result<String> {
    let bpfctl_cmd = Command::cargo_bin("bpfctl")?;
    let bpfctl_path = bpfctl_cmd.get_program().to_str().unwrap();

    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => args.extend([
            "--image-url",
            URETPROBE_IMAGE_LOC,
            "--pull-policy",
            "Always",
        ]),
        LoadType::File => args.extend(["-s", "my_uretprobe", "--path", URETPROBE_FILE_LOC]),
    }

    args.extend(["uprobe", "-f", "main", "-t", bpfctl_path, "-r"]);

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added uretprobe program: {:?} from: {:?}",
        uuid, load_type
    );
    Ok(uuid.to_string())
}

/// Install a kprobe program with bpfctl
pub fn add_kprobe(globals: Option<Vec<&str>>, load_type: &LoadType) -> Result<String> {
    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => {
            args.extend(["--image-url", KPROBE_IMAGE_LOC, "--pull-policy", "Always"])
        }
        LoadType::File => args.extend(["-s", "my_kprobe", "--path", KPROBE_FILE_LOC]),
    }

    args.extend(["kprobe", "-f", "try_to_wake_up"]);

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added kprobe program: {:?} from: {:?}",
        uuid, load_type
    );
    Ok(uuid.to_string())
}

/// Install a kretprobe program with bpfctl
pub fn add_kretprobe(globals: Option<Vec<&str>>, load_type: &LoadType) -> Result<String> {
    let mut args = Vec::new();

    match load_type {
        LoadType::Image => {
            args.push("load-from-image");
        }
        LoadType::File => {
            args.push("load-from-file");
        }
    }

    if let Some(g) = globals {
        args.push("--global");
        args.extend(g);
    }

    match load_type {
        LoadType::Image => args.extend([
            "--image-url",
            KRETPROBE_IMAGE_LOC,
            "--pull-policy",
            "Always",
        ]),
        LoadType::File => args.extend(["-s", "my_kretprobe", "--path", KRETPROBE_FILE_LOC]),
    }

    args.extend(["kprobe", "-f", "try_to_wake_up", "-r"]);

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    let uuid = stdout.trim();
    assert!(!uuid.is_empty());
    debug!(
        "Successfully added kretprobe program: {:?} from: {:?}",
        uuid, load_type
    );
    Ok(uuid.to_string())
}

/// Delete a bpfd program using bpfctl
pub fn bpfd_del_program(uuid: &str) {
    Command::cargo_bin("bpfctl")
        .unwrap()
        .args(["unload", uuid.trim()])
        .assert()
        .success()
        .stdout(is_empty());

    debug!("Successfully deleted program: \"{}\"", uuid.trim());
}

/// Retrieve the output of bpfctl list
pub fn bpfd_list() -> Result<String> {
    let output = Command::cargo_bin("bpfctl")?.args(["list"]).ok();
    let stdout = String::from_utf8(output.unwrap().stdout);
    Ok(stdout.unwrap())
}

pub fn bpfd_pull_bytecode() -> Result<String> {
    let mut args = vec!["pull-bytecode"];

    args.extend([
        "--image-url",
        TRACEPOINT_IMAGE_LOC,
        "--pull-policy",
        "Always",
    ]);

    let output = Command::cargo_bin("bpfctl")?.args(args).ok();
    let stdout = String::from_utf8(output.unwrap().stdout);
    Ok(stdout.unwrap())
}

pub fn get_image_path() -> PathBuf {
    let relative_path = str::replace(TRACEPOINT_IMAGE_LOC, ":", "/");
    Path::new(BYTECODE_IMAGE_CONTENT_STORE).join(relative_path)
}

/// Retrieve the output of bpfctl list
pub fn tc_filter_list(iface: &str) -> Result<String> {
    let output = Command::new("tc")
        .args(["filter", "show", "dev", iface, "ingress"])
        .output()
        .expect("tc filter show dev lo ingress");
    let stdout = String::from_utf8(output.unwrap().stdout);
    Ok(stdout.unwrap())
}

/// Verify the specified interface exists
/// TODO: make OS agnostic (network-interface crate https://lib.rs/crates/network-interface?)
pub fn iface_exists(bpfd_iface: &str) -> bool {
    let output = Command::new("ip")
        .args(["link", "show"])
        .output()
        .expect("ip link show");
    let link_out = String::from_utf8(output.stdout).unwrap();

    if link_out.contains(bpfd_iface) {
        return true;
    }

    false
}

pub struct NamespaceGuard {
    name: &'static str,
}

/// Delete namespace.  This causes the associated veth's and ip's to also get
/// deleted
fn delete_namespace(name: &'static str) {
    let status = Command::new("ip")
        .args(["netns", "delete", name])
        .status()
        .expect("could not delete namespace");

    if !status.success() {
        println!("could not delete namespace {name}: {status}");
    } else {
        debug!("namespace {} deleted", name);
    }
}

impl Drop for NamespaceGuard {
    fn drop(&mut self) {
        delete_namespace(self.name);
    }
}

fn get_ip_prefix() -> String {
    match option_env!("BPFD_IP_PREFIX") {
        Some(ip_prefix) => ip_prefix.to_string(),
        None => DEFAULT_IP_PREFIX.to_string(),
    }
}

fn get_ip_addr(id: &str) -> String {
    format!("{}.{}", get_ip_prefix(), id)
}

fn ip_prefix_exists(prefix: &String) -> bool {
    // It sometimes takes the previous delete_namespace(NS_NAME) a little time to clean
    // everything up, so give it a little time before checking.
    sleep(Duration::from_millis(100));

    let output = Command::new("ip")
        .args(["address", "list"])
        .output()
        .expect("Failed to create namespace");

    if !output.status.success() {
        panic!("could not execute \"ip address list\" command");
    };

    let stdout = String::from_utf8(output.unwrap().stdout).unwrap();
    stdout.contains(prefix)
}

/// Create a namespace [`NS_NAME`] with a veth pair and IP addresses
pub fn create_namespace() -> Result<NamespaceGuard> {
    if ip_prefix_exists(&get_ip_prefix()) {
        panic!(
            "ip prefix {} is in use, specify an available prefix with env BPFD_IP_PREFIX.",
            get_ip_prefix()
        );
    }

    let status = Command::new("ip")
        .args(["netns", "add", NS_NAME])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        panic!("failed to create namespace {NS_NAME}: {status}");
    }

    let status = Command::new("ip")
        .args([
            "link", "add", HOST_VETH, "type", "veth", "peer", "name", NS_VETH,
        ])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        delete_namespace(NS_NAME);
        panic!("failed to create veth pair {HOST_VETH}-{NS_VETH}: {status}");
    }

    let status = Command::new("ip")
        .args(["link", "set", NS_VETH, "netns", NS_NAME])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        delete_namespace(NS_NAME);
        panic!("failed to add veth {NS_VETH} to {NS_NAME}: {status}");
    }

    let ns_ip_mask = format!("{}/{}", get_ip_addr(NS_IP_ID), IP_MASK);

    let status = Command::new("ip")
        .args([
            "netns",
            "exec",
            NS_NAME,
            "ip",
            "addr",
            "add",
            &ns_ip_mask,
            "dev",
            NS_VETH,
        ])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        delete_namespace(NS_NAME);
        panic!(
            "failed to add ip address {ns_ip_mask} to {NS_VETH}: {status}\n
        if {ns_ip_mask} is not available, specify a usable prefix with env BPFD_IT_PREFIX.\n
        for example: export BPFD_IT_PREFIX=\"192.168.1\""
        );
    }

    let host_ip_mask = format!("{}/{}", get_ip_addr(HOST_IP_ID), IP_MASK);

    let status = Command::new("ip")
        .args(["addr", "add", &host_ip_mask, "dev", HOST_VETH])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        delete_namespace(NS_NAME);
        panic!("failed to add ip address {ns_ip_mask} to {HOST_VETH}: {status}");
    }

    let status = Command::new("ip")
        .args([
            "netns", "exec", NS_NAME, "ip", "link", "set", "dev", NS_VETH, "up",
        ])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        delete_namespace(NS_NAME);
        panic!("failed to set dev {NS_VETH} to up: {status}");
    }

    let status = Command::new("ip")
        .args(["link", "set", "dev", HOST_VETH, "up"])
        .status()
        .expect("Failed to create namespace");

    if !status.success() {
        delete_namespace(NS_NAME);
        panic!("failed to set dev {HOST_VETH} to up: {status}");
    }

    debug!("Successfully created namespace {NS_NAME}");

    Ok(NamespaceGuard { name: NS_NAME })
}

/// start a ping to the network namespace IP address with output logged to [`PING_FILE_NAME`]
pub fn start_ping() -> Result<ChildGuard> {
    let f = File::create(PING_FILE_NAME).unwrap();
    let ping_process = Command::new("ping")
        .args([get_ip_addr(NS_IP_ID)])
        .stdout(std::process::Stdio::from(f))
        .spawn()
        .map(|c| ChildGuard {
            name: "ping",
            child: c,
        })
        .expect("Failed to start ping");

    debug!(
        "sucessfully started ping to namespace {} at address {}",
        NS_NAME,
        get_ip_addr(NS_IP_ID),
    );

    Ok(ping_process)
}

/// Get the ping log from [`PING_FILE_NAME`]
pub fn read_ping_log() -> Result<String> {
    let mut f = File::open(PING_FILE_NAME)?;
    let mut buffer = String::new();
    f.read_to_string(&mut buffer)?;
    Ok(buffer)
}

/// start sending /sys/kernel/debug/tracing/trace_pipe to [`TRACE_PIPE_FILE_NAME`]
pub fn start_trace_pipe() -> Result<ChildGuard> {
    // The trace_pipe is clear on read, so we start a process to read it to
    // clear any logs left over from the last test.  Kill that process and then
    // start the real one.

    // Start it
    let f = File::create(TRACE_PIPE_FILE_NAME).unwrap();
    let mut trace_process = Command::new("cat")
        .args(["/sys/kernel/debug/tracing/trace_pipe"])
        .stdout(std::process::Stdio::from(f))
        .spawn()
        .expect("Failed to start trace_pipe");

    sleep(Duration::from_secs(1));

    // Kill it
    if let Err(e) = trace_process.kill() {
        println!("Could not kill trace_pipe: {e}");
    }
    if let Err(e) = trace_process.wait() {
        println!("Could not wait for trace_pipe: {e}");
    }

    // Start it again
    let f = File::create(TRACE_PIPE_FILE_NAME).unwrap();
    let trace_process = Command::new("cat")
        .args(["/sys/kernel/debug/tracing/trace_pipe"])
        .stdout(std::process::Stdio::from(f))
        .spawn()
        .map(|c| ChildGuard {
            name: "trace_pipe",
            child: c,
        })
        .expect("Failed to start trace_pipe");

    debug!("sucessfully started cat trace_pipe",);
    Ok(trace_process)
}

/// get the trace_pipe output from [`TRACE_PIPE_FILE_NAME`]
pub fn read_trace_pipe_log() -> Result<String> {
    let mut f = File::open(TRACE_PIPE_FILE_NAME)?;
    let mut buffer = String::new();
    f.read_to_string(&mut buffer)?;
    debug!("trace_pipe output read to string");
    Ok(buffer)
}

/// Verify that the programs in the uuid list have been loaded.  Then delete them
/// and verify that they have been deleted.
pub fn verify_and_delete_programs(uuids: Vec<String>) {
    // Verify bpfctl list contains the uuids of each program
    let bpfctl_list = bpfd_list().unwrap();
    for id in uuids.iter() {
        assert!(bpfctl_list.contains(id.trim()));
    }

    // Delete the installed programs
    debug!("Deleting bpfd program");
    for id in uuids.iter() {
        bpfd_del_program(id)
    }

    // Verify bpfctl list does not contain the uuids of the deleted programs
    // and that there are no panics if bpfctl does not contain any programs.
    let bpfctl_list = bpfd_list().unwrap();
    for id in uuids.iter() {
        assert!(!bpfctl_list.contains(id.trim()));
    }
}

/// Returns true if the bpffs has entries and false if it doesn't
pub fn bpffs_has_entries(path: &str) -> bool {
    PathBuf::from(path).read_dir().unwrap().next().is_some()
}
