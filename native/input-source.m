// macOS Text Input Sources for the nn and nnime workflow objects.
#import <AppKit/AppKit.h>
#import <Carbon/Carbon.h>
#import <CoreGraphics/CoreGraphics.h>

static NSURL *preferenceURL(void) {
    NSString *path = NSProcessInfo.processInfo.environment[@"alfred_workflow_data"];
    if (!path.length) {
        path = [NSHomeDirectory() stringByAppendingPathComponent:
            @"Library/Application Support/Alfred/Workflow Data/com.inchans.alfnaversearchplus"];
    }
    return [[NSURL fileURLWithPath:path isDirectory:YES] URLByAppendingPathComponent:@"input-source"];
}

static NSString *property(TISInputSourceRef source, CFStringRef key) {
    return source ? (__bridge NSString *)TISGetInputSourceProperty(source, key) : nil;
}

static NSArray *enabledSources(void) {
    NSDictionary *filter = @{
        (__bridge NSString *)kTISPropertyInputSourceCategory: (__bridge NSString *)kTISCategoryKeyboardInputSource,
        (__bridge NSString *)kTISPropertyInputSourceIsSelectCapable: @YES
    };
    return CFBridgingRelease(TISCreateInputSourceList((__bridge CFDictionaryRef)filter, false));
}

static TISInputSourceRef findSource(NSArray *sources, NSString *identifier) {
    for (id entry in sources) {
        TISInputSourceRef source = (__bridge TISInputSourceRef)entry;
        if ([property(source, kTISPropertyInputSourceID) isEqualToString:identifier]) return source;
    }
    return NULL;
}

static BOOL alfredPanelIsActive(void) {
    NSString *owner = NSWorkspace.sharedWorkspace.frontmostApplication.bundleIdentifier;
    NSString *alfred = @"com.runningwithcrayons.Alfred";
    if ([owner isEqualToString:alfred] || [owner isEqualToString:@"com.runningwithcrayons.Alfred-UI"]) return YES;

    // The embedded UI takes keyboard input in a nonactivating Alfred panel.
    // The app behind it still owns the menu bar. Only inspect window metadata.
    NSRunningApplication *host = [NSRunningApplication runningApplicationsWithBundleIdentifier:alfred].firstObject;
    if (!host) return NO;
    NSArray *windows = CFBridgingRelease(CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID));
    for (NSDictionary *window in windows) {
        if ([window[(__bridge NSString *)kCGWindowOwnerPID] intValue] != host.processIdentifier ||
            [window[(__bridge NSString *)kCGWindowLayer] integerValue] <= NSNormalWindowLevel ||
            [window[(__bridge NSString *)kCGWindowAlpha] doubleValue] <= 0) continue;
        CGRect bounds;
        if (CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)window[(__bridge NSString *)kCGWindowBounds], &bounds) &&
            CGRectGetHeight(bounds) > NSStatusBar.systemStatusBar.thickness) return YES;
    }
    return NO;
}

static NSDictionary *choices(NSString *preference, NSString *query) {
    NSMutableArray *items = [NSMutableArray array];
    NSArray *sources = enabledSources();
    BOOL available = preference.length && findSource(sources, preference);
    NSString *subtitle = preference.length && !available
        ? @"저장한 입력 소스를 사용할 수 없어 현재 입력기를 사용합니다."
        : @"Alfred를 열었을 때의 입력기를 사용합니다.";
    [items addObject:@{@"title": available ? @"현재 입력 소스 유지" : @"✓ 현재 입력 소스 유지",
                       @"subtitle":subtitle, @"arg":@"keep"}];
    for (id entry in sources) {
        TISInputSourceRef source = (__bridge TISInputSourceRef)entry;
        NSString *identifier = property(source, kTISPropertyInputSourceID);
        NSString *name = property(source, kTISPropertyLocalizedName) ?: identifier;
        if (!identifier.length) continue;
        NSString *title = [identifier isEqualToString:preference] ? [@"✓ " stringByAppendingString:name] : name;
        [items addObject:@{@"title":title, @"subtitle":identifier, @"arg":identifier}];
    }
    if (query.length) {
        NSPredicate *match = [NSPredicate predicateWithBlock:^BOOL(NSDictionary *item, NSDictionary *bindings) {
            return [item[@"title"] localizedCaseInsensitiveContainsString:query] ||
                   [item[@"arg"] localizedCaseInsensitiveContainsString:query];
        }];
        [items filterUsingPredicate:match];
    }
    return @{@"items":items};
}

static NSDictionary *enterSearch(NSString *preferred) {
    if (!preferred.length) return @{};
    // Called synchronously by nn's preload, while its new panel may still be hidden.
    id current = CFBridgingRelease(TISCopyCurrentKeyboardInputSource());
    NSString *previous = property((__bridge TISInputSourceRef)current, kTISPropertyInputSourceID);
    if (!previous.length || [previous isEqualToString:preferred]) return @{};
    NSArray *sources = enabledSources();
    TISInputSourceRef target = findSource(sources, preferred);
    if (!target) return @{};
    if (TISSelectInputSource(target) != noErr) return @{};
    // macOS publishes the new ID before a composing IME accepts its first key.
    // Give the input method time to attach before Alfred presents the field.
    if (![property(target, kTISPropertyInputSourceType) isEqual:(__bridge NSString *)kTISTypeKeyboardLayout]) {
        [NSThread sleepForTimeInterval:0.15];
    }
    return @{@"variables":@{@"nnime_previous":previous, @"nnime_applied":preferred}};
}

static void leaveSearch(NSString *previous, NSString *applied) {
    if (!previous.length || !applied.length) return;
    NSArray *sources = enabledSources();
    TISInputSourceRef target = findSource(sources, previous);
    id current = CFBridgingRelease(TISCopyCurrentKeyboardInputSource());
    if (target && [property((__bridge TISInputSourceRef)current, kTISPropertyInputSourceID)
                   isEqualToString:applied] && alfredPanelIsActive()) TISSelectInputSource(target);
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        NSArray<NSString *> *args = NSProcessInfo.processInfo.arguments;
        if (argc < 2) return 1;
        NSString *cmd = args[1];
        NSString *arg = argc > 2 ? args[2] : @"";
        NSURL *file = preferenceURL();
        NSDictionary *output = @{};
        if ([cmd isEqualToString:@"list"] || [cmd isEqualToString:@"enter"]) {
            NSString *preferred = [NSString stringWithContentsOfURL:file encoding:NSUTF8StringEncoding error:NULL];
            output = [cmd isEqualToString:@"list"] ? choices(preferred, arg) : enterSearch(preferred);
        } else if ([cmd isEqualToString:@"set"]) {
            BOOL keep = [arg isEqualToString:@"keep"];
            if (!keep && !findSource(enabledSources(), arg)) return 1;
            NSError *error = nil;
            if (![[NSFileManager defaultManager] createDirectoryAtURL:file.URLByDeletingLastPathComponent
                    withIntermediateDirectories:YES attributes:nil error:&error] ||
                ![(keep ? @"" : arg) writeToURL:file atomically:YES encoding:NSUTF8StringEncoding error:&error]) {
                fprintf(stderr, "%s\n", error.localizedDescription.UTF8String);
                return 1;
            }
            return 0;
        } else if ([cmd isEqualToString:@"leave"] && argc == 4) {
            leaveSearch(arg, args[3]);
            output = @{@"variables":@{@"nnime_previous":@"", @"nnime_applied":@""}};
        } else {
            return 1;
        }
        NSData *json = [NSJSONSerialization dataWithJSONObject:output options:0 error:NULL];
        if (!json) return 1;
        fwrite(json.bytes, 1, json.length, stdout);
        return 0;
    }
}
