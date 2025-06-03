#!/bin/bash

set -euo pipefail

# Configuration
REPO_NAME="${REPO_NAME:-}"  # Will be auto-detected from go.mod if empty
SOURCE_PATH_IN_REPO="${SOURCE_PATH_IN_REPO:-schemas/api/v1}"
BUF_YAML_PATH="${BUF_YAML_PATH:-buf.yaml}"
GO_MOD_PATH="${GO_MOD_PATH:-../go.mod}"
PROTO_FILE_NAME="${PROTO_FILE_NAME:-}"  # Specific proto file to download (optional)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Function to parse go.mod and extract protobuf libraries
parse_protobuf_libraries_from_gomod() {
    local go_mod_path="$1"
    
    if [[ ! -f "$go_mod_path" ]]; then
        log_error "go.mod file not found at: $go_mod_path"
        exit 1
    fi
    
    log_info "Parsing protobuf libraries from $go_mod_path..."
    
    # Find the "// Protobuf libraries" comment and extract replace directives that follow
    local protobuf_libs=()
    local found_comment=false
    
    while IFS= read -r line; do
        # Check if we found the protobuf libraries comment
        if [[ "$line" =~ ^[[:space:]]*//[[:space:]]*Protobuf[[:space:]]+libraries ]]; then
            found_comment=true
            continue
        fi
        
        # If we found the comment, look for replace directives
        if [[ "$found_comment" == true ]]; then
            # Stop at empty lines or other comments that might indicate end of section
            if [[ -z "${line// }" ]] || [[ "$line" =~ ^[[:space:]]*$ ]]; then
                break
            fi
            
            # Parse replace directive: replace local-module version => github.com/org/repo version
            if [[ "$line" =~ ^[[:space:]]*replace[[:space:]]+([^[:space:]]+)[[:space:]]+([^[:space:]]+)[[:space:]]*=\>[[:space:]]*([^[:space:]]+)[[:space:]]+([^[:space:]]+) ]]; then
                local local_module="${BASH_REMATCH[1]}"
                local local_version="${BASH_REMATCH[2]}"
                local remote_repo="${BASH_REMATCH[3]}"
                local remote_version="${BASH_REMATCH[4]}"
                
                # Extract just the repository path (remove github.com/ prefix if present)
                if [[ "$remote_repo" =~ ^github\.com/(.+)$ ]]; then
                    remote_repo="${BASH_REMATCH[1]}"
                fi
                
                protobuf_libs+=("github.com/$remote_repo:$remote_version")
                log_info "Found protobuf library: github.com/$remote_repo@$remote_version"
            fi
        fi
    done < "$go_mod_path"
    
    if [[ "$found_comment" == false ]]; then
        log_error "Could not find '// Protobuf libraries' comment in $go_mod_path"
        exit 1
    fi
    
    if [[ ${#protobuf_libs[@]} -eq 0 ]]; then
        log_warning "No protobuf libraries found after '// Protobuf libraries' comment"
        exit 1
    fi
    
    # Return the array as newline-separated values
    printf '%s\n' "${protobuf_libs[@]}"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

A flexible script to download and update proto files from remote repositories.
Automatically detects protobuf libraries from go.mod using // Protobuf libraries comment or allows manual specification.

Options:
    -h, --help              Show this help message
    -v, --version VERSION   Specify version to download (default: auto-detect from go.mod)
    -r, --repo REPO         Repository name (default: auto-detect from go.mod)
    -s, --source PATH       Source path in repository (default: schemas/api/v1)
    -b, --buf-yaml PATH     Path to buf.yaml file (default: buf.yaml)
    -g, --go-mod PATH       Path to go.mod file (default: ../go.mod)
    -f, --proto-file FILE   Download only specific proto file (e.g., product_availability.proto)
    -d, --dry-run          Show what would be done without executing
    --list-versions        List available versions for all repos and exit
    --single-repo          Process only the first repository found

Environment Variables:
    REPO_NAME              Repository name (overrides auto-detection)
    SOURCE_PATH_IN_REPO    Source path in repository
    BUF_YAML_PATH          Path to buf.yaml file
    GO_MOD_PATH            Path to go.mod file
    PROTO_FILE_NAME        Specific proto file to download

Examples:
    $0                                          # Auto-detect and download from go.mod
    $0 --version v0.12.0 --single-repo        # Download specific version of first repo
    $0 --repo github.com/my-org/my-api         # Use specific repository
    $0 --proto-file product_availability.proto # Download only product_availability.proto
    $0 --dry-run                               # Preview what would be done
    $0 --list-versions                         # List available versions for all repos

EOF
}

# Function to parse buf.yaml and extract module path
get_module_path_from_buf_yaml() {
    local buf_yaml="$1"
    
    if [[ ! -f "$buf_yaml" ]]; then
        log_error "buf.yaml file not found at: $buf_yaml"
        exit 1
    fi
    
    # Extract the first module path from buf.yaml
    local module_path
    module_path=$(grep -A 1 "modules:" "$buf_yaml" | grep "path:" | head -1 | sed 's/.*path: *//' | tr -d '"' | tr -d "'")
    
    if [[ -z "$module_path" ]]; then
        log_error "Could not find module path in $buf_yaml"
        exit 1
    fi
    
    echo "$module_path"
}

# Function to get latest version from a Go module
get_latest_version() {
    local repo="$1"
    
    log_info "Checking latest version for $repo..."
    
    # Use go list to get the latest version
    local latest_version
    if latest_version=$(go list -m -versions "$repo" 2>/dev/null | awk '{print $NF}'); then
        if [[ -n "$latest_version" && "$latest_version" != "$repo" ]]; then
            echo "$latest_version"
        else
            # Fallback: try to get latest from go proxy
            local encoded_repo
            encoded_repo=$(echo "$repo" | sed 's|/|%2F|g')
            local proxy_url="https://proxy.golang.org/$encoded_repo/@latest"
            
            if command -v curl >/dev/null 2>&1; then
                latest_version=$(curl -s "$proxy_url" | grep -o '"Version":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "")
            elif command -v wget >/dev/null 2>&1; then
                latest_version=$(wget -qO- "$proxy_url" | grep -o '"Version":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "")
            fi
            
            if [[ -n "$latest_version" ]]; then
                echo "$latest_version"
            else
                log_error "Could not determine latest version for $repo"
                exit 1
            fi
        fi
    else
        log_error "Could not fetch version information for $repo"
        exit 1
    fi
}

# Function to list available versions
list_versions() {
    local repo="$1"
    
    log_info "Listing available versions for $repo..."
    
    # Try to get versions using go list
    if go list -m -versions "$repo" 2>/dev/null; then
        return
    fi
    
    # Fallback: try to get from go proxy
    local encoded_repo
    encoded_repo=$(echo "$repo" | sed 's|/|%2F|g')
    local proxy_url="https://proxy.golang.org/$encoded_repo/@v/list"
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "$proxy_url" 2>/dev/null || log_error "Could not fetch versions"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$proxy_url" 2>/dev/null || log_error "Could not fetch versions"
    else
        log_error "Neither curl nor wget available to fetch versions"
        exit 1
    fi
}

# Function to download and extract proto files
download_and_copy_protos() {
    local repo="$1"
    local version="$2"
    local source_path="$3"
    local target_path="$4"
    local dry_run="$5"
    local specific_file="$6"  # Optional parameter for specific proto file
    
    local repo_with_version="${repo}@${version}"
    local gomodcache
    gomodcache=$(go env GOMODCACHE)
    
    if [[ -z "$gomodcache" ]]; then
        log_error "Could not determine GOMODCACHE"
        exit 1
    fi
    
    local source_dir="${gomodcache}/${repo_with_version}/${source_path}"
    
    if [[ "$dry_run" == "true" ]]; then
        log_info "DRY RUN MODE - Actions that would be performed:"
        echo "  1. Download: go mod download $repo_with_version"
        echo "  2. Source directory: $source_dir"
        echo "  3. Target directory: $target_path"
        
        if [[ -d "$source_dir" ]]; then
            if [[ -n "$specific_file" ]]; then
                echo "  4. Specific proto file that would be copied:"
                if [[ -f "$source_dir/$specific_file" ]]; then
                    echo "     - $specific_file"
                else
                    echo "     - $specific_file (NOT FOUND - would fail)"
                fi
            else
                echo "  4. Proto files that would be copied:"
                find "$source_dir" -name "*.proto" -type f | while read -r file; do
                    echo "     - $(basename "$file")"
                done
            fi
        else
            echo "  4. Source directory does not exist yet (would be created by download)"
        fi
        return
    fi
    
    log_info "Downloading $repo_with_version..."
    if ! go mod download "$repo_with_version"; then
        log_error "Failed to download $repo_with_version"
        exit 1
    fi
    
    if [[ ! -d "$source_dir" ]]; then
        log_error "Source directory not found after download: $source_dir"
        exit 1
    fi
    
    # Create target directory if it doesn't exist
    if [[ ! -d "$target_path" ]]; then
        log_info "Creating target directory: $target_path"
        mkdir -p "$target_path"
    fi
    
    # Handle specific file or all proto files
    if [[ -n "$specific_file" ]]; then
        # Copy only the specific proto file
        local source_file="$source_dir/$specific_file"
        local target_file="$target_path/$specific_file"
        
        if [[ ! -f "$source_file" ]]; then
            log_error "Specific proto file not found: $source_file"
            log_info "Available proto files in $source_dir:"
            find "$source_dir" -name "*.proto" -type f | while read -r file; do
                echo "  - $(basename "$file")"
            done
            exit 1
        fi
        
        # Make target file writable if it exists
        if [[ -f "$target_file" ]]; then
            chmod u+w "$target_file"
        fi
        
        log_info "Copying specific proto file: $specific_file"
        cp "$source_file" "$target_path/"
        log_success "Successfully copied proto file: $specific_file"
    else
        # Copy all proto files (original behavior)
        local proto_count
        proto_count=$(find "$source_dir" -name "*.proto" -type f | wc -l | tr -d ' ')
        
        if [[ "$proto_count" -eq 0 ]]; then
            log_warning "No .proto files found in $source_dir"
            return
        fi
        
        log_info "Copying $proto_count proto file(s) from $source_dir to $target_path..."
        
        # Make all existing proto files writable before copying
        find "$target_path" -name "*.proto" -type f -exec chmod u+w {} \;
        
        # Copy all proto files
        find "$source_dir" -name "*.proto" -type f -exec cp {} "$target_path/" \;
        
        log_success "Successfully copied proto files:"
        find "$target_path" -name "*.proto" -type f | while read -r file; do
            echo "  - $(basename "$file")"
        done
    fi
}

# Function to process a single repository
process_repository() {
    local repo_info="$1"
    local target_path="$2"
    local source_path="$3"
    local dry_run="$4"
    local specified_version="$5"
    
    # Parse repo_info (format: github.com/org/repo:version)
    local repo="${repo_info%:*}"
    local default_version="${repo_info#*:}"
    
    # Use specified version if provided, otherwise use the version from go.mod
    local version="${specified_version:-$default_version}"
    
    log_info "Processing repository: $repo"
    
    # Download and copy proto files
    download_and_copy_protos "$repo" "$version" "$source_path" "$target_path" "$dry_run" "$PROTO_FILE_NAME"
}

# Main function
main() {
    local version=""
    local repo="$REPO_NAME"
    local source_path="$SOURCE_PATH_IN_REPO"
    local buf_yaml="$BUF_YAML_PATH"
    local go_mod_path="$GO_MOD_PATH"
    local dry_run="false"
    local list_versions_flag="false"
    local single_repo_flag="false"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--version)
                version="$2"
                shift 2
                ;;
            -r|--repo)
                repo="$2"
                shift 2
                ;;
            -s|--source)
                source_path="$2"
                shift 2
                ;;
            -b|--buf-yaml)
                buf_yaml="$2"
                shift 2
                ;;
            -g|--go-mod)
                go_mod_path="$2"
                shift 2
                ;;
            -f|--proto-file)
                PROTO_FILE_NAME="$2"
                shift 2
                ;;
            -d|--dry-run)
                dry_run="true"
                shift
                ;;
            --list-versions)
                list_versions_flag="true"
                shift
                ;;
            --single-repo)
                single_repo_flag="true"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Validate required tools
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is required but not installed"
        exit 1
    fi
    
    # Get target path from buf.yaml
    local target_path
    target_path=$(get_module_path_from_buf_yaml "$buf_yaml")
    log_info "Target path from $buf_yaml: $target_path"
    
    # Determine repositories to process
    local repositories=()
    if [[ -n "$repo" ]]; then
        # Single repository specified via command line or environment
        if [[ -z "$version" ]]; then
            version=$(get_latest_version "$repo")
            log_info "Latest version detected for $repo: $version"
        fi
        repositories=("$repo:$version")
    else
        # Auto-detect from go.mod
        log_info "Auto-detecting protobuf libraries from $go_mod_path..."
        # Use a loop instead of readarray for better compatibility
        while IFS= read -r line; do
            repositories+=("$line")
        done < <(parse_protobuf_libraries_from_gomod "$go_mod_path")
    fi
    
    # List versions for all repositories if requested
    if [[ "$list_versions_flag" == "true" ]]; then
        for repo_info in "${repositories[@]}"; do
            local repo_name="${repo_info%:*}"
            echo "--- Versions for $repo_name ---"
            list_versions "$repo_name"
            echo ""
        done
        exit 0
    fi
    
    # Process repositories
    if [[ "$single_repo_flag" == "true" && ${#repositories[@]} -gt 1 ]]; then
        log_info "Single repo mode: processing only the first repository"
        repositories=("${repositories[0]}")
    fi
    
    log_info "Processing ${#repositories[@]} repository(ies)..."
    
    for repo_info in "${repositories[@]}"; do
        process_repository "$repo_info" "$target_path" "$source_path" "$dry_run" "$version"
        
        if [[ "$dry_run" != "true" ]]; then
            echo ""  # Add spacing between repositories
        fi
    done
    
    if [[ "$dry_run" != "true" ]]; then
        log_success "All proto files updated successfully!"
        log_info "You may want to run 'buf generate' to regenerate code from the updated protos"
    fi
}

# Run main function with all arguments
main "$@" 