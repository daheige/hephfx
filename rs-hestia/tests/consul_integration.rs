//! Consul integration tests.
//!
//! These tests are ignored by default because they require a running Consul agent
//! at `127.0.0.1:8500`. Start one with:
//!
//! ```text
//! docker run -d --name consul \
//!   -p 8500:8500 \
//!   hashicorp/consul consul agent -dev -ui -client=0.0.0.0
//! ```

use env_logger::Target;
use rs_hestia::consul::{Options, new_discovery, new_registry, new_resolver_builder};
use rs_hestia::{Context, Service};

fn init_logger() {
    env_logger::builder().target(Target::Stdout).try_init().ok();
}

fn endpoints() -> Vec<String> {
    vec!["http://127.0.0.1:8500".to_string()]
}

#[tokio::test]
#[ignore = "requires local consul on 127.0.0.1:8500"]
async fn test_registry_and_discovery() {
    init_logger();
    let ctx = Context::new();
    let registry = new_registry(Options::new(endpoints()))
        .await
        .expect("create registry");

    let mut svc = Service {
        network: "tcp".to_string(),
        name: "my-test".to_string(),
        address: "127.0.0.1:18080".to_string(),
        version: "v1".to_string(),
        ..Default::default()
    };

    registry
        .register(&ctx, &mut svc)
        .await
        .expect("register service");
    assert!(!svc.instance_id.is_empty());
    assert_eq!(svc.weight, 100);
    assert!(svc.healthy);

    // Give Consul a moment to mark the TTL check passing.
    tokio::time::sleep(tokio::time::Duration::from_secs(3)).await;

    let discovery = new_discovery(Options::new(endpoints()))
        .await
        .expect("create discovery");
    let services = discovery
        .get_services(&ctx, "my-test", "v1")
        .await
        .expect("get services");
    assert!(!services.is_empty());
    assert!(services.iter().any(|s| s.instance_id == svc.instance_id));

    registry
        .deregister(&ctx, &mut svc)
        .await
        .expect("deregister service");
    assert!(!svc.healthy);
}

#[tokio::test]
#[ignore = "requires local consul on 127.0.0.1:8500"]
async fn test_register() {
    init_logger();
    let ctx = Context::new();
    let registry = new_registry(Options::new(endpoints()))
        .await
        .expect("create registry");

    let mut svc = Service {
        network: "tcp".to_string(),
        name: "my-test".to_string(),
        address: "127.0.0.1:18080".to_string(),
        version: "v1".to_string(),
        ..Default::default()
    };

    registry
        .register(&ctx, &mut svc)
        .await
        .expect("register service");
    assert!(!svc.instance_id.is_empty());
    assert_eq!(svc.weight, 100);
    assert!(svc.healthy);

    // Give Consul a moment to mark the TTL check passing.
    tokio::time::sleep(tokio::time::Duration::from_secs(30)).await;

    registry
        .deregister(&ctx, &mut svc)
        .await
        .expect("deregister service");
    assert!(!svc.healthy);
}

#[tokio::test]
#[ignore = "requires local consul on 127.0.0.1:8500"]
async fn test_discovery(){
    let ctx = Context::new();
    let discovery = new_discovery(Options::new(endpoints()))
        .await
        .expect("create discovery");
    let services = discovery
        .get_services(&ctx, "my-test", "v1")
        .await
        .expect("get services");
    assert!(!services.is_empty());
    println!("{:#?}", services);
}

#[tokio::test]
#[ignore = "requires local consul on 127.0.0.1:8500"]
async fn test_discovery_watch() {
    init_logger();
    let ctx = Context::new();
    let registry = new_registry(Options::new(endpoints()))
        .await
        .expect("create registry");
    let discovery = new_discovery(Options::new(endpoints()).with_enable_watch())
        .await
        .expect("create discovery with watch");

    let mut svc = Service {
        name: "watch-test".to_string(),
        address: "127.0.0.1:18081".to_string(),
        version: "v1".to_string(),
        ..Default::default()
    };

    registry
        .register(&ctx, &mut svc)
        .await
        .expect("register service");

    tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;

    // First fetch triggers the watch.
    let services = discovery
        .get_services(&ctx, "watch-test", "v1")
        .await
        .expect("get services");
    assert!(!services.is_empty());

    // Allow watch task to start.
    tokio::time::sleep(tokio::time::Duration::from_millis(200)).await;

    registry
        .deregister(&ctx, &mut svc)
        .await
        .expect("deregister service");

    // Wait for the watch to reflect the removal.
    tokio::time::sleep(tokio::time::Duration::from_millis(1500)).await;

    let services = discovery.get_services(&ctx, "watch-test", "v1").await;
    assert!(services.is_err() || services.unwrap().is_empty());
}

#[tokio::test]
#[ignore = "requires local consul on 127.0.0.1:8500"]
async fn test_resolver_build() {
    init_logger();
    let ctx = Context::new();
    let registry = new_registry(Options::new(endpoints()))
        .await
        .expect("create registry");
    let discovery = new_discovery(Options::new(endpoints()))
        .await
        .expect("create discovery");

    let mut svc = Service {
        name: "resolver-test".to_string(),
        address: "127.0.0.1:18082".to_string(),
        version: "v1".to_string(),
        protocol: rs_hestia::ProtocolType::Grpc,
        ..Default::default()
    };

    registry
        .register(&ctx, &mut svc)
        .await
        .expect("register service");

    tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;

    let builder = new_resolver_builder(discovery);
    let _channel = builder
        .build("consul:///resolver-test/v1")
        .await
        .expect("build channel");

    registry
        .deregister(&ctx, &mut svc)
        .await
        .expect("deregister service");
}

// cargo test --test consul_integration test_registry -- --nocapture
#[tokio::test]
#[ignore = "requires local consul on 127.0.0.1:8500"]
async fn test_registry() {
    init_logger();
    let ctx = Context::new();
    let registry = new_registry(
        Options::new(vec!["http://127.0.0.1:8500".to_string()])
            .with_prefix("/services".to_string()),
    )
    .await
    .expect("create registry");

    let mut svc = Service {
        name: "resolver-test".to_string(),
        address: "127.0.0.1:18082".to_string(),
        version: "v1".to_string(),
        protocol: rs_hestia::ProtocolType::Grpc,
        ..Default::default()
    };

    registry
        .register(&ctx, &mut svc)
        .await
        .expect("register service");

    // Keep the service registered for manual inspection.
    tokio::time::sleep(tokio::time::Duration::from_secs(100)).await;

    registry
        .deregister(&ctx, &mut svc)
        .await
        .expect("deregister service");
}
