namespace go thriftapi

struct Location {
  1: double lat
  2: double lng
}

struct Order {
  1: string id
  2: string userId
  3: Location origin
  4: Location destination
  5: string status
  6: optional string assignedDroneId
  7: optional Location handoffOrigin
  8: i64 createdAt
  9: i64 updatedAt
  10: optional i64 reservedAt
  11: optional i64 pickedUpAt
  12: optional i64 deliveredAt
  13: optional i64 failedAt
  14: optional string failureReason
}

struct OrderView {
  1: Order order
  2: optional Location currentLocation
  3: optional i64 etaSeconds
}

struct Drone {
  1: string id
  2: string status
  3: optional Location lastLocation
  4: optional i64 lastHeartbeatAt
  5: optional string currentOrderId
  6: i64 createdAt
  7: i64 updatedAt
}

struct DroneStatus {
  1: Drone drone
  2: optional OrderView currentOrder
}

struct TokenRequest {
  1: string name
  2: string role
}

struct TokenResponse {
  1: string token
  2: i64 expiresAt
}

struct SubmitOrderRequest {
  1: string authToken
  2: Location origin
  3: Location destination
}

struct OrderIDRequest {
  1: string authToken
  2: string orderId
}

struct AuthRequest {
  1: string authToken
}

struct FailOrderRequest {
  1: string authToken
  2: string orderId
  3: string reason
}

struct HeartbeatRequest {
  1: string authToken
  2: Location location
}

struct ListOrdersRequest {
  1: string authToken
  2: optional string status
  3: optional i32 limit
  4: optional i32 offset
}

struct UpdateOrderRequest {
  1: string authToken
  2: string orderId
  3: optional Location origin
  4: optional Location destination
}

struct DroneIDRequest {
  1: string authToken
  2: string droneId
}

service AuthService {
  TokenResponse IssueToken(1: TokenRequest request)
}

service OrderService {
  Order SubmitOrder(1: SubmitOrderRequest request)
  Order WithdrawOrder(1: OrderIDRequest request)
  OrderView GetOrder(1: OrderIDRequest request)
}

service DroneService {
  Order ReserveJob(1: AuthRequest request)
  Order PickupOrder(1: OrderIDRequest request)
  Order DeliverOrder(1: OrderIDRequest request)
  Order FailOrder(1: FailOrderRequest request)
  Drone MarkBroken(1: AuthRequest request)
  DroneStatus Heartbeat(1: HeartbeatRequest request)
  OrderView CurrentOrder(1: AuthRequest request)
}

service AdminService {
  list<OrderView> ListOrders(1: ListOrdersRequest request)
  Order UpdateOrder(1: UpdateOrderRequest request)
  list<Drone> ListDrones(1: AuthRequest request)
  Drone MarkDroneBroken(1: DroneIDRequest request)
  Drone MarkDroneFixed(1: DroneIDRequest request)
}
