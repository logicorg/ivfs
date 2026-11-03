import { Link } from "@tanstack/react-router"
import {
  ArrowDownLeft,
  ArrowLeftRight,
  ArrowUpRight,
  ChartLine,
  Coins,
  LayoutDashboard,
  LogOut,
  Settings,
  Tags,
  TrendingUp,
  Users,
  Wallet,
  type LucideIcon,
} from "lucide-react"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { logoutPath } from "@/lib/session"

// The screens, in the order the sidebar lists them. Each later batch adds
// its own.
const screens = [
  { to: "/", label: "Overview", icon: LayoutDashboard },
  { to: "/net-worth", label: "Net worth", icon: TrendingUp },
  { to: "/accounts", label: "Accounts", icon: Wallet },
  { to: "/investments", label: "Investments", icon: ChartLine },
  { to: "/categories", label: "Categories", icon: Tags },
  { to: "/counterparties", label: "Counterparties", icon: Users },
  { to: "/currencies", label: "Currencies", icon: Coins },
  { to: "/settings", label: "Settings", icon: Settings },
] as const

// The transactions recorded most often, one page each, so they are a click
// away from any screen.
const recordings = [
  { to: "/record/expense", label: "Expense", icon: ArrowUpRight },
  { to: "/record/income", label: "Income", icon: ArrowDownLeft },
  { to: "/record/transfer", label: "Transfer", icon: ArrowLeftRight },
] as const

export function AppSidebar() {
  return (
    <Sidebar>
      <SidebarHeader className="px-4 py-3 font-semibold">ivfs</SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <ScreenLinks links={screens} />
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup>
          <SidebarGroupLabel>Record</SidebarGroupLabel>
          <SidebarGroupContent>
            <ScreenLinks links={recordings} />
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        {/* A form, not a fetch: the gateway answers with a redirect to the
            provider's sign-out, which only a navigation follows. */}
        <form method="post" action={logoutPath}>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton type="submit" tooltip="Sign out">
                <LogOut />
                <span>Sign out</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </form>
      </SidebarFooter>
    </Sidebar>
  )
}

type ScreenLink = (typeof screens)[number]["to"] | (typeof recordings)[number]["to"]

function ScreenLinks({ links }: { links: readonly { to: ScreenLink; label: string; icon: LucideIcon }[] }) {
  return (
    <SidebarMenu>
      {links.map(({ to, label, icon: Icon }) => (
        <SidebarMenuItem key={to}>
          <SidebarMenuButton asChild tooltip={label}>
            <Link to={to} activeProps={{ "data-active": true }}>
              <Icon />
              <span>{label}</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  )
}
