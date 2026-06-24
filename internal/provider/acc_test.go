package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const testNamePrefix = "tftest"

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"k0s": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccClusterResource_SingleNode(t *testing.T) {
	name := testNamePrefix + "-sn-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig(name, "v1.32.2-k0s.0", true, 1, 0),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("single_node"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact("v1.32.2-k0s.0"),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("image"),
						knownvalue.StringExact("docker.io/k0sproject/k0s:v1.32.2-k0s.0"),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_MultiNode(t *testing.T) {
	name := testNamePrefix + "-mn-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig(name, "v1.32.2-k0s.0", false, 1, 1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("single_node"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("controller_count"),
						knownvalue.Int64Exact(1),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("worker_count"),
						knownvalue.Int64Exact(1),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccClusterDataSource_SingleNode(t *testing.T) {
	name := testNamePrefix + "-ds-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfigWithDataSource(name, "v1.32.2-k0s.0"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_Import(t *testing.T) {
	name := testNamePrefix + "-imp-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig(name, "v1.32.2-k0s.0", true, 1, 0),
			},
			{
				ResourceName:                         "k0s_cluster.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"kubeconfig"},
			},
		},
	})
}

// --- critical path tests ---------------------------------------------------

func TestAccClusterResource_WaitForReadyFalse(t *testing.T) {
	name := testNamePrefix + "-nw-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfigWaitForReadyFalse(name, "v1.32.2-k0s.0"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("wait_for_ready"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_KubeconfigPath(t *testing.T) {
	name := testNamePrefix + "-kp-" + randomSuffix()
	kubeconfigPath := t.TempDir() + "/" + name + "/kubeconfig.yaml"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfigKubeconfigPath(name, "v1.32.2-k0s.0", kubeconfigPath),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig_path"),
						knownvalue.StringExact(kubeconfigPath),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_MultiWorker(t *testing.T) {
	name := testNamePrefix + "-mw-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig(name, "v1.32.2-k0s.0", false, 1, 2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("single_node"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("worker_count"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccClusterDataSource_MultiNode(t *testing.T) {
	name := testNamePrefix + "-dm-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfigWithDataSourceMultiNode(name, "v1.32.2-k0s.0"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("running"),
					),
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.k0s_cluster.test",
						tfjsonpath.New("single_node"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_ImportMultiNode(t *testing.T) {
	name := testNamePrefix + "-im-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig(name, "v1.32.2-k0s.0", false, 1, 1),
			},
			{
				ResourceName:                         "k0s_cluster.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"kubeconfig"},
			},
		},
	})
}

func TestAccClusterVersionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "k0s_versions" "test" {}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.k0s_versions.test",
						tfjsonpath.New("versions"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.k0s_versions.test",
						tfjsonpath.New("latest"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

// --- feature coverage tests ------------------------------------------------

func TestAccClusterResource_ExtraArgs(t *testing.T) {
	name := testNamePrefix + "-ea-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfigExtraArgs(name, "v1.32.2-k0s.0"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_TimeoutsBlock(t *testing.T) {
	name := testNamePrefix + "-to-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfigTimeoutsBlock(name, "v1.32.2-k0s.0"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccClusterResource_ComputedOutputs(t *testing.T) {
	name := testNamePrefix + "-co-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckClusterDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccClusterConfig(name, "v1.32.2-k0s.0", true, 1, 0),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"k0s_cluster.test",
						tfjsonpath.New("kubeconfig"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

// --- helpers ---------------------------------------------------------------

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests require TF_ACC=1")
	}

	docker := newDockerClient()
	_, err := docker.run(context.Background(), "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		t.Fatalf("Docker must be available for acceptance tests: %s", err)
	}
}

func testAccClusterConfig(name, version string, singleNode bool, controllers, workers int) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name             = %[1]q\n"+
		"version          = %[2]q\n"+
		"single_node      = %[3]t\n"+
		"controller_count = %[4]d\n"+
		"worker_count     = %[5]d\n"+
		"}\n", name, version, singleNode, controllers, workers)
}

func testAccClusterConfigWaitForReadyFalse(name, version string) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name            = %[1]q\n"+
		"version         = %[2]q\n"+
		"single_node     = true\n"+
		"wait_for_ready  = false\n"+
		"}\n", name, version)
}

func testAccClusterConfigKubeconfigPath(name, version, kubeconfigPath string) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name             = %[1]q\n"+
		"version          = %[2]q\n"+
		"single_node      = true\n"+
		"kubeconfig_path  = %[3]q\n"+
		"}\n", name, version, kubeconfigPath)
}

func testAccClusterConfigExtraArgs(name, version string) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name            = %[1]q\n"+
		"version         = %[2]q\n"+
		"single_node     = true\n"+
		"extra_args      = [\"--disable-components\", \"coredns\"]\n"+
		"}\n", name, version)
}

func testAccClusterConfigTimeoutsBlock(name, version string) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name       = %[1]q\n"+
		"version    = %[2]q\n"+
		"single_node = true\n"+
		"timeouts = {\n"+
		"  create = \"180s\"\n"+
		"  read   = \"60s\"\n"+
		"  delete = \"120s\"\n"+
		"}\n"+
		"}\n", name, version)
}

func testAccClusterConfigWithDataSource(name, version string) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name       = %[1]q\n"+
		"version    = %[2]q\n"+
		"single_node = true\n"+
		"}\n"+
		"data \"k0s_cluster\" \"test\" {\n"+
		"name = k0s_cluster.test.name\n"+
		"}\n", name, version)
}

func testAccClusterConfigWithDataSourceMultiNode(name, version string) string {
	return fmt.Sprintf("resource \"k0s_cluster\" \"test\" {\n"+
		"name             = %[1]q\n"+
		"version          = %[2]q\n"+
		"single_node      = false\n"+
		"controller_count = 1\n"+
		"worker_count     = 1\n"+
		"}\n"+
		"data \"k0s_cluster\" \"test\" {\n"+
		"name = k0s_cluster.test.name\n"+
		"}\n", name, version)
}

func testAccCheckClusterDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		docker := newDockerClient()
		ctx := context.Background()

		for _, cn := range []string{name, controllerName(name, 1), workerName(name, 1)} {
			running, err := docker.isRunning(ctx, cn)
			if err != nil {
				return fmt.Errorf("error checking container %q: %w", cn, err)
			}
			if running {
				return fmt.Errorf("container %q still exists after destroy", cn)
			}
		}

		netName := networkName(name)
		exists, err := docker.networkExists(ctx, netName)
		if err != nil {
			return fmt.Errorf("error checking network %q: %w", netName, err)
		}
		if exists {
			return fmt.Errorf("network %q still exists after destroy", netName)
		}

		return nil
	}
}

func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "test"
	}
	return hex.EncodeToString(b)
}
