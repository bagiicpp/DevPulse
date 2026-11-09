import {
  ChartNoAxesCombined,
  FolderGit2,
  House,
  LayoutDashboard,
  LogIn,
  MessageSquare,
  PanelLeftClose,
  PanelLeftOpen,
  Settings2,
  UserRound,
} from "lucide-react"
import { NavLink, useLocation } from "react-router"
import type { AuthState } from "@/hooks/use-session"

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
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar"

const navigation = [
  { title: "Home", href: "/", icon: House },
  { title: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { title: "Repositories", href: "/repositories", icon: FolderGit2 },
  { title: "Project Analysis", href: "/analysis", icon: ChartNoAxesCombined },
  { title: "Chat with AI", href: "/chat", icon: MessageSquare },
]

const accountNavigation = [
  { title: "Account", href: "/account", icon: UserRound },
  { title: "Settings", href: "/settings", icon: Settings2 },
]

function SidebarBrand() {
  const { state, isMobile, openMobile, toggleSidebar, setOpenMobile } =
    useSidebar()
  const expanded = isMobile ? openMobile : state === "expanded"

  return (
    <div className="flex h-12 items-center justify-between gap-2 px-2">
      <NavLink
        to="/"
        className="group-data-[collapsible=icon]:hidden"
        onClick={() => setOpenMobile(false)}
      >
        <span className="font-serif text-xl font-semibold italic tracking-tight">
          DevPulse
        </span>
      </NavLink>

      <button
        type="button"
        onClick={toggleSidebar}
        aria-label="Toggle sidebar"
        aria-expanded={expanded}
        className="flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
      >
        {expanded ? (
          <PanelLeftClose className="size-4" />
        ) : (
          <PanelLeftOpen className="size-4" />
        )}
      </button>
    </div>
  )
}

export function AppSidebar({ auth }: { auth: AuthState }) {
  const { pathname } = useLocation()
  const { setOpenMobile } = useSidebar()

  return (
    <>
      <SidebarHeader className="border-b bg-sidebar px-2 py-1 md:hidden">
        <SidebarBrand />
      </SidebarHeader>
      <Sidebar collapsible="icon">
        <SidebarHeader className="border-b px-2 py-1">
          <SidebarBrand />
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Workspace</SidebarGroupLabel>

            <SidebarGroupContent>
              <SidebarMenu>
                {navigation.map((item) => (
                  <SidebarMenuItem key={item.href}>
                    <SidebarMenuButton
                      render={<NavLink to={item.href} end />}
                      isActive={pathname === item.href}
                      tooltip={item.title}
                      aria-label={item.title}
                      onClick={() => setOpenMobile(false)}
                    >
                      <item.icon aria-hidden="true" />
                      <span>{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter className="border-t p-2">
          <SidebarMenu>
            {accountNavigation.map((item) => (
              <SidebarMenuItem key={item.href}>
                <SidebarMenuButton
                  render={<NavLink to={item.href} />}
                  isActive={pathname === item.href}
                  tooltip={item.title}
                  aria-label={item.title}
                  onClick={() => setOpenMobile(false)}
                >
                  <item.icon aria-hidden="true" />
                  <span>{item.title}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
            {!auth.loading && !auth.user && (
              <SidebarMenuItem>
                <SidebarMenuButton
                  render={<NavLink to="/login" />}
                  tooltip="Sign in"
                  aria-label="Sign in"
                  onClick={() => setOpenMobile(false)}
                >
                  <LogIn aria-hidden="true" />
                  <span>Sign in</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )}
          </SidebarMenu>
          {auth.user && (
            <p className="truncate px-2 pt-2 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
              Signed in as @{auth.user.login}
            </p>
          )}
        </SidebarFooter>

        <SidebarRail />
      </Sidebar>
    </>
  )
}
