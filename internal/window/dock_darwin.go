//go:build darwin

package window

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework AppKit
#include <AppKit/AppKit.h>
#include <dispatch/dispatch.h>
static void voidConfigureLauncherSpaces(void) {
 dispatch_async(dispatch_get_main_queue(), ^{
  for (NSWindow *window in [NSApp windows]) {
   if ([[window title] isEqualToString:@"Void"]) {
    [window setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorFullScreenAuxiliary];
   }
  }
 });
}
*/
import "C"
import "os"

// Keep the launcher reachable on the current Space without changing the
// application's activation policy while a window is being shown.
func configureLauncherSpaces() {
	if os.Getenv("KEEL_HEADLESS") != "1" {
		C.voidConfigureLauncherSpaces()
	}
}
