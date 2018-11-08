#pragma once

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace ServiceControl {

// The function type to cancel an in-flight request.
using CancelFunc = std::function<void()>;

}  // namespace ServiceControl
}  // namespace HttpFilters
}  // namespace Extensions
}  // namespace Envoy
